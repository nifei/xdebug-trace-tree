package xdebugtrace

import (
	"fmt"
	"io"
	"os"
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"time"
	"strings"
)

type XDebugTrace struct {
	version string
	format int64
	start_ts time.Time
	functions map[int]XDebugFunction
}

type XDebugFunction struct {
	depth int
	timeEnter float64
	memoryEnter int64
	name string
	internal string
	file string
	line string
	params []string
	timeExit float64
	memoryExit int64
	timeDiff float64
	memoryDiff int64
	ret string
	lines []string
}

func (f XDebugFunction) str() string{
	return fmt.Sprintf("depth:%d;name:%s;\ttimeEnter:%f;memoryEnter:%d;internal:%s;file:%s;line:%s;params:%s;timeExit:%f;memoryExit:%d;timeDiff:%f;memoryDiff:%d;ret:%s",
		f.depth, f.name, f.timeEnter, f.memoryEnter, f.internal, f.file, f.line,
		f.params, f.timeExit, f.memoryExit, f.timeDiff, f.memoryDiff, f.ret)
}

func newTrace () (x XDebugTrace) {
	x.functions = make(map[int]XDebugFunction)
	return
}

func ParseContent(lines []string) *XDebugTrace{
	x := newTrace()
	if len(lines) < 4 {
		return &x
	}
	// Version: 2.4.0
	x.version = ParseVersion(lines[0])
	// File format: 4
	x.format, _ = strconv.ParseInt(ParseFormat(lines[1]), 10 ,8)
	// "TRACE START [2017-03-14 14:34:51]"
	x.start_ts = ParseStartTime(lines[3])
	for _, line := range lines[3:] {
		x.ParseLine(line)
	}
	return &x
}

func (x *XDebugTrace) ParseLine (line string) XDebugFunction {
	// fmt.Println(line)
	parts := strings.Split(line, "\t")
	if len(parts) < 5 {
		return XDebugFunction{}
	}
	return x.ParseFunction(parts)
}

func (x *XDebugTrace) ParseFunction (parts []string) (XDebugFunction) {
	var f XDebugFunction
	nr := toInt(parts[1])
	typ := parts[2]
	switch typ {
	case "0": // enter
		f.depth, f.timeEnter, f.memoryEnter, f.name, f.internal, f.file, f.line = toInt(parts[0]),
			toFloat(parts[3]), toInt64(parts[4]), parts[5], parts[6], parts[8], parts[9]
		if len(parts[7])>0 {
			f.params = []string{parts[7]}
		} else if len(parts)>=12{
			f.params = parts[11:]
		}
		break
	case "1": // exit
		f = x.functions[nr]
		f.timeExit, f.memoryExit = toFloat(parts[3]), toInt64(parts[4])
		f.timeDiff = f.timeExit - f.timeEnter
		f.memoryDiff = f.memoryExit - f.memoryEnter
		break
	case "R": // return
		f = x.functions[nr]
		f.ret = parts[5]
		break
	default:
		return f
	}
	x.functions[nr] = f
	return f
}

type HtmlLines struct {
	lines []string
}

func (h *HtmlLines) add (line string) {
	h.lines = append(h.lines, line)
}

func (x *XDebugTrace) ToHtml() (lines []string) {
	var h HtmlLines
	headLines := []string{"<head>",
		"<link rel=\"stylesheet\" href=\"style.css\" type=\"text/css\">",
		"</head>"}
	for _, hl := range headLines {
		h.add(hl)
	}
	scriptLines := []string {"<script type=\"text/javascript\" src=\"script.js\"></script>"}
	for _, sl := range scriptLines {
		h.add(sl)
	}

	h.add("<div class=\"f header\">");
	h.add("<div class=\"func\">Function Call</div>");
	h.add("<div class=\"data\">");
	h.add("<span class=\"file\">File:Line</span>");
	h.add("<span class=\"timediff\">ΔTime</span>");
	h.add("<span class=\"memorydiff\">ΔMemory</span>");
	h.add("<span class=\"time\">Time</span>");
	h.add("</div>");
	h.add("</div>");

	level := 0
	for _, f := range x.functions {
		if f.depth > level {
			for i:=level; i<f.depth; i++ {
				h.add("<div class=\"d\">")
			}
		} else if f.depth < level {
			for i:=f.depth;i<level;i++ {
				h.add("</div>")
			}
		}
		level = f.depth
		class := "f"
		if (f.internal == "1") {
			class = class + " i"
		}
		h.add("<div class=\"" + class + "\">")

		h.add("<div class=\"func\">")
		h.add("<span class=\"name\">" + f.name + "</span>")

		h.add("<span class=\"params short\">" + join(",", f.params) +  "</span>")
		if len(f.ret) > 0 {
			h.add("→ <span class=\"return short\">" + f.ret + "</span>")
		}

		h.add("</div>");
		h.add("<div class=\"data\">")
		h.add("<span class=\"file\" title=\"" +
			f.file + ":" + f.line + "\">" +
			f.file + ":" + f.line+"</span>")

		h.add("<span class=\"timediff\">" + strconv.FormatFloat(f.timeDiff, 'g', 1, 64) + "</span>");

		h.add("<span class=\"memorydiff\">" + string(f.memoryDiff) +  "</span>");
		h.add("<span class=\"time\">" + strconv.FormatFloat(f.timeEnter, 'g', 1, 64) +  "</span>");
		h.add("</div>");
		h.add("</div>");
	}
	if level > 0 {
		for i:=0;i<level;i++ {
			h.add("</div>")
		}
	}
	return h.lines
}

func writeFile(filename string, lines []string) {
	// open output file
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()
	w := bufio.NewWriter(fo)

	for  _, line := range lines{
		// write a chunk
		if _, err := w.WriteString(line + "\n"); err != nil {
			panic(err)
		}
	}

	if err = w.Flush(); err != nil {
		panic(err)
	}
}

func join(sep string, ss []string)(s string) {
	if len(ss) == 0 {
		return
	}
	s = ss[0]
	for i:=1;i<len(ss);i++ {
		s = s + ss[i]
	}
	return
}

func toFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
func toInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func toInt(s string) int {
	return int(toInt64(s))
}

func ParseVersion(line string) string {
	pattern := `Version: \d*.\d*.\d*`
	reg := regexp.MustCompile(pattern)
	return reg.FindString(line)[len("Version: "):]
}

func ParseFormat(line string) string {
	pattern := `File format: \d*`
	reg := regexp.MustCompile(pattern)
	return reg.FindString(line)[len("File format: "):]
}

func ParseStartTime(line string) time.Time {
	head := "TRACE START ["
	tsformat := "2006-01-02 15:04:05"
	tsstring := line[len(head):len(head)+len(tsformat)]
	timestamp:= toTime(tsstring)
	return timestamp
}

func toTime(line string) time.Time {
	tsformat := "2006-01-02 15:04:05"
	timestamp, _:= time.Parse(tsformat, line)
	return timestamp
}

func ParseFile(filename string) *XDebugTrace {
	_, lines := readFileWithReadLine(filename)
	return ParseContent(lines)
}


func readFileWithReadLine(fn string) (err error, lines []string) {
	file, err := os.Open(fn)
	defer file.Close()

	if err != nil {
		return err, lines
	}

	reader := bufio.NewReader(file)

	for {
		var buffer bytes.Buffer

		var l []byte
		var isPrefix bool
		for {
			l, isPrefix, err = reader.ReadLine()
			buffer.Write(l)

			if !isPrefix {
				break
			}

			if err != nil {
				break
			}
		}

		if err == io.EOF {
			break
		}

		line := buffer.String()
		// fmt.Println("read line:" + line)
		lines = append(lines, line)
	}

	if err != io.EOF {
		fmt.Printf(" > Failed!: %v\n", err)
	}

	return err,  lines
}
