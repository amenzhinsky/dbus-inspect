package main

import (
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

var (
	qFlag       bool
	systemFlag  bool
	indentFlag  string
	noColorFlag bool
	sigsFlag    bool
	valuesFlag  bool
	stdinFlag   bool

	methodsFlag    bool
	propertiesFlag bool
	signalsFlag    bool
)

var errUsage = errors.New("invalid usage")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dbus-inspect [flag...] <destination> [path...]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.BoolVar(&qFlag, "q", false, "provide short overview, destination names or paths only")
	flag.BoolVar(&systemFlag, "system", false, "connect to the system bus instead of the session one")
	flag.StringVar(&indentFlag, "indent", "  ", "set indentation string")
	flag.BoolVar(&noColorFlag, "no-color", false, "disable color in output text")
	flag.BoolVar(&sigsFlag, "signatures", false, "show argument signatures instead of human-readable types")
	flag.BoolVar(&methodsFlag, "methods", false, "show only methods")
	flag.BoolVar(&propertiesFlag, "props", false, "show only properties")
	flag.BoolVar(&signalsFlag, "signals", false, "show only signals")
	flag.BoolVar(&valuesFlag, "values", false, "read property values")
	flag.BoolVar(&stdinFlag, "stdin", false, "introspect stdin")
	flag.Parse()

	if err := run(os.Stdout); err != nil {
		if err == errUsage {
			flag.Usage()
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run(w io.Writer) error {
	if stdinFlag {
		if valuesFlag {
			return errors.New("cannot combine -values flag with -stdin")
		}
		return introspectStdin(w)
	}

	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	if flag.NArg() == 0 {
		return listNames(c)
	}
	if flag.NArg() > 2 {
		return errUsage
	}

	dest := flag.Arg(0)
	path := dbus.ObjectPath("/")
	recursive := true
	if flag.NArg() > 1 {
		recursive = false
		path = dbus.ObjectPath(flag.Arg(1))
	}
	if err = introspectNode(w, c, dest, path, recursive); err != nil {
		return err
	}
	return nil
}

func introspectStdin(w io.Writer) error {
	var node introspect.Node
	if err := xml.NewDecoder(os.Stdin).Decode(&node); err != nil {
		return err
	}
	return printNode(w, &node, nil, indent(0))
}

func connect() (*dbus.Conn, error) {
	if systemFlag {
		return dbus.SystemBus()
	}
	return dbus.SessionBus()
}

func introspectNode(
	w io.Writer, c *dbus.Conn,
	dest string, path dbus.ObjectPath, recursive bool,
) error {
	var s string
	if err := c.Object(dest, path).Call(
		"org.freedesktop.DBus.Introspectable.Introspect", 0,
	).Store(&s); err != nil {
		return err
	}
	var node introspect.Node
	if err := xml.Unmarshal([]byte(s), &node); err != nil {
		return err
	}
	if qFlag {
		fmt.Fprintln(w, string(path))
		goto Children
	}
	fmt.Fprintln(w, color(string(path), termBold))
	if err := printNode(w, &node, c.Object(dest, path), indent(1)); err != nil {
		return err
	}
	if !recursive {
		return nil
	}
Children:
	for _, child := range node.Children {
		next := dbus.ObjectPath("/" + child.Name)
		if path != "/" {
			next = path + next
		}
		if err := introspectNode(w, c, dest, next, recursive); err != nil {
			return err
		}
	}
	return nil
}

func sectionEnabled(v bool) bool {
	return (!methodsFlag && !propertiesFlag && !signalsFlag) || v
}

func printNode(w io.Writer, node *introspect.Node, o dbus.BusObject, indent func(int) string) error {
	for _, iface := range node.Interfaces {
		fmt.Fprintln(w, indent(0)+color(iface.Name, termGreen))
		if len(iface.Methods) > 0 && sectionEnabled(methodsFlag) {
			fmt.Fprintln(w, indent(1)+color("Methods", termYellow))
			for _, method := range iface.Methods {
				for _, annotation := range method.Annotations {
					fmt.Fprintf(w, "%s%s\n", indent(2), fmtAnnotation(annotation))
				}
				in, out, err := methodArgs(method.Args)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "%s%s(%s) → (%s)\n", indent(2), method.Name, fmtArgs(in), fmtArgs(out))
			}
		}
		if len(iface.Properties) > 0 && sectionEnabled(propertiesFlag) {
			fmt.Fprintln(w, indent(1)+color("Properties", termYellow))
			for _, property := range iface.Properties {
				for _, annotation := range property.Annotations {
					fmt.Fprintf(w, "%s%s\n", indent(2), fmtAnnotation(annotation))
				}
				fmt.Fprintf(w, "%s%s %s %s",
					indent(2),
					property.Name,
					fmtArgType(property.Type),
					color("["+property.Access+"]", termGray),
				)

				// o is nil when we're reading xml definition from a file
				if o == nil || !valuesFlag {
					fmt.Fprintln(w)
				} else {
					var v interface{}
					if err := o.Call(
						"org.freedesktop.DBus.Properties.Get", 0, iface.Name, property.Name,
					).Store(&v); err != nil {
						v = color("(error: "+err.Error()+")", termRed)
					}
					fmt.Fprintf(w, " = %v\n", v)
				}
			}
		}
		if len(iface.Signals) > 0 && sectionEnabled(signalsFlag) {
			fmt.Fprintln(w, indent(1)+color("Signals", termYellow))
			for _, signal := range iface.Signals {
				for _, annotation := range signal.Annotations {
					fmt.Fprintf(w, "%s%s\n", indent(2), fmtAnnotation(annotation))
				}
				fmt.Fprintf(w, "%s%s(%s)\n", indent(2), signal.Name, fmtArgs(signal.Args))
			}
		}
	}
	return nil
}

func indent(n int) func(int) string {
	return func(i int) string {
		return strings.Repeat(indentFlag, i+n)
	}
}

const (
	termBold    = "1"
	termRed     = "31"
	termGreen   = "32"
	termYellow  = "33"
	termBlue    = "34"
	termMagenta = "35"
	termGray    = "90"
)

func color(s string, nums ...string) string {
	if len(nums) == 0 {
		panic("no colors given")
	}
	if noColorFlag {
		return s
	}
	r := strings.Builder{}
	for _, num := range nums {
		r.WriteString("\x1b[" + num + "m")
	}
	r.WriteString(s)
	r.WriteString("\x1b[0m")
	return r.String()
}

func listNames(c *dbus.Conn) error {
	var dests []string
	if err := c.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&dests); err != nil {
		return err
	}
	// move down names that start with ':'
	sort.Slice(dests, func(i, j int) bool {
		if dests[i][0] == ':' && dests[j][0] != ':' {
			return false
		}
		if dests[i][0] != ':' && dests[j][0] == ':' {
			return true
		}
		return dests[i] < dests[j]
	})
	for _, dest := range dests {
		if qFlag {
			fmt.Println(dest)
			continue
		}
		var pid uint32
		if err := c.BusObject().Call(
			"org.freedesktop.DBus.GetConnectionUnixProcessID", 0, dest,
		).Store(&pid); err != nil {
			return err
		}
		fmt.Printf("%s %s %s\n",
			color(dest, termBold),
			color(fmt.Sprintf("%d", pid), termBlue),
			color(cmdline(pid), termGray),
		)
	}
	return nil
}

func cmdline(pid uint32) string {
	b, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	cmdline := strings.ReplaceAll(string(b), "\x00", " ")
	return cmdline[:len(cmdline)-1]
}

func methodArgs(args []introspect.Arg) ([]introspect.Arg, []introspect.Arg, error) {
	var in, out []introspect.Arg
	for _, arg := range args {
		switch arg.Direction {
		case "in":
			in = append(in, arg)
		case "out":
			out = append(out, arg)
		default:
			return nil, nil, fmt.Errorf("unknown arg direction: %q", arg.Direction)
		}
	}
	return in, out, nil
}

func fmtArgs(args []introspect.Arg) string {
	s := make([]string, 0, len(args))
	for i, arg := range args {
		s = append(s, fmtArg(i, arg))
	}
	return strings.Join(s, ", ")
}

func fmtArg(i int, arg introspect.Arg) string {
	name := arg.Name
	if name == "" {
		name = "arg_" + strconv.Itoa(i)
	}
	return color(name, termGray) + " " + fmtArgType(arg.Type)
}

func fmtArgType(s string) string {
	if sigsFlag {
		return color(s, termBlue)
	}
	return color(fmtSig(s), termBlue)
}

func fmtAnnotation(annotation introspect.Annotation) string {
	return color(fmt.Sprintf("@%s = %s", annotation.Name, annotation.Value), termMagenta)
}

func fmtSig(sig string) string {
	var r []string
	var l int
	for {
		s, rlen := next(sig[l:])
		if rlen == 0 {
			return "Malformed(" + sig + ")"
		}

		r = append(r, s)
		l += rlen
		if len(sig) == l {
			return strings.Join(r, ", ")
		}
	}
}

func next(sig string) (string, int) {
	if len(sig) == 0 {
		return "", 0
	}
	switch sig[0] {
	case 'y':
		return "Byte", 1
	case 'b':
		return "Bool", 1
	case 'n':
		return "Int16", 1
	case 'q':
		return "Uint16", 1
	case 'i':
		return "Int32", 1
	case 'u':
		return "Uint32", 1
	case 'x':
		return "Int64", 1
	case 't':
		return "Uint64", 1
	case 'd':
		return "Double", 1
	case 'h':
		return "UnixFD", 1
	case 's':
		return "String", 1
	case 'o':
		return "Object", 1
	case 'v':
		return "Variant", 1
	case 'g':
		return "Signature", 1
	case 'a':
		if sig[1] == '{' { // dictionary
			i := 4
			k, rlen := next(sig[2:])
			if rlen != 1 {
				// key is not a primitive
				return "", 0
			}
			v, rlen := next(sig[3:])
			if rlen == 0 {
				// value is not available
				return "", 0
			}
			i += rlen
			return "Dict{" + k + ", " + v + "}", i
		}
		s, rlen := next(sig[1:])
		return "Array[" + s + "]", rlen + 1
	case '(':
		i := 1
		n := 1
		for i < len(sig) && n != 0 {
			if sig[i] == '(' {
				n++
			} else if sig[i] == ')' {
				n--
			}
			i++
		}
		return "Struct(" + structFields(sig[1:i-1]) + ")", i
	default:
		return "Unknown(" + string(sig[0]) + ")", 1
	}
}

func structFields(sig string) string {
	fields := make([]string, 0, len(sig))
	for i := 0; i < len(sig); {
		s, rlen := next(sig[i:])
		if rlen == 0 {
			break
		}
		i += rlen
		fields = append(fields, s)
	}
	return strings.Join(fields, ", ")
}
