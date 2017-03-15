package xdebugtrace

import (
	"testing"
	"io"
	"strings"
	"fmt"
)


func Test_parseFile(t *testing.T) {
	filename := "./xdebug-trace.xt"
	x := ParseFile(filename)
	for i, f := range x.functions {
		fmt.Println(f.str())
		if len(f.name) == 0 {
			t.Fatal(i, f.str())
		}
	}
	if len(x.functions) != 6601 {
		t.Fatal(len(x.functions))
	}
}

func Test_ToHtml(t *testing.T) {
	filename := "./xdebug-trace.xt"
	x := ParseFile(filename)
	for _, l := range x.ToHtml() {
		fmt.Println(l)
	}
	writeFile(filename + ".html", x.ToHtml())
}

func Test_ParseFunctionNotExist(t *testing.T) {
	line := "1	0	0	0.000301	369752	{main}	1		./index.php	0	0"
	parts := strings.Split(line, "\t")
	x := newTrace()
	function := x.ParseFunction(parts)
	if function.depth != 1 {
		t.Fatal(function)
	}
}

func Test_invalidLine(t *testing.T) {
	lines := []string {"TRACE END   [2017-03-14 14:34:51]",
		"0.162800	552"}
	x := newTrace()
	x.ParseLine(lines[0])
	x.ParseLine(lines[1])
	if len(x.functions) > 0 {
		t.Fatal(x)
	}
}

func Test_ParseFunctionLong(t *testing.T) {
	line := "2	1	0	0.000318	369752	define	0		./index.php	20	2	'ENVIRONMENT'	'production'"
	x := newTrace()
	x.ParseLine(line)
	if len(x.functions) != 1 {
		t.Fatal(x)
	}
	if (x.functions[1].name != "define") {
		t.Fatal(x.functions[1].str())
	}
}

func Test_ParseFunctionPair(t *testing.T) {
	lines := []string{
		"2	1	0	0.000318	369752	define	0		./index.php	20	2	'ENVIRONMENT'	'production'",
		"2	1	1	0.000327	369784",
		"2	1	R			TRUE"}
	x := newTrace()
	x.ParseLine(lines[0])
	x.ParseLine(lines[1])
	x.ParseLine(lines[2])
	f1 := x.functions[1]
	fmt.Println(f1.str())
	if f1.name != "define" {
		t.Fatal(f1.name)
	}
	if f1.depth != 2 {
		t.Fatal(f1.depth)
	}
	if f1.line != "20" {
		t.Fatal(f1.line)
	}
	if f1.file != "./index.php" {
		t.Fatal(f1.file)
	}
	if f1.memoryDiff != 32 {
		t.Fatal(f1.memoryDiff)
	}
	if len(f1.params) != 2 {
		t.Fatal(f1.params)
	}
}


func Test_readFileWithReadLine(t *testing.T) {
	filename := "xdebug-trace.xt"
	err, lines := readFileWithReadLine(filename)
	if err != io.EOF {
		t.Fatal("err:", err)
	}
	if len(lines) == 0 {
		t.Fail()
	}
}