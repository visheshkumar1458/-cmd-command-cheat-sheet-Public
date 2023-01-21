// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package cmd_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/juju/gnuflag"
	"github.com/juju/loggo"
	gitjujutesting "github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"github.com/juju/cmd/v3"
	"github.com/juju/cmd/v3/cmdtesting"
)

func initDefenestrate(args []string) (*cmd.SuperCommand, *TestCommand, error) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{Name: "jujutest"})
	tc := &TestCommand{Name: "defenestrate"}
	jc.Register(tc)
	return jc, tc, cmdtesting.InitCommand(jc, args)
}

func initDefenestrateWithAliases(c *gc.C, args []string) (*cmd.SuperCommand, *TestCommand, error) {
	dir := c.MkDir()
	filename := filepath.Join(dir, "aliases")
	err := ioutil.WriteFile(filename, []byte(`
def = defenestrate
be-firm = defenestrate --option firmly
other = missing 
		`), 0644)
	c.Assert(err, gc.IsNil)
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{Name: "jujutest", UserAliasesFilename: filename})
	tc := &TestCommand{Name: "defenestrate"}
	jc.Register(tc)
	return jc, tc, cmdtesting.InitCommand(jc, args)
}

type SuperCommandSuite struct {
	gitjujutesting.IsolationSuite

	ctx *cmd.Context
}

var _ = gc.Suite(&SuperCommandSuite{})

const docText = "\n    documentation\\s+- Generate the documentation for all commands"
const helpText = "\n    help\\s+- Show help on a command or other topic."
const helpCommandsText = "commands:" + docText + helpText

func (s *SuperCommandSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)
	s.ctx = cmdtesting.Context(c)
	loggo.ReplaceDefaultWriter(cmd.NewWarningWriter(s.ctx.Stderr))
}

func (s *SuperCommandSuite) TestDispatch(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{Name: "jujutest"})
	info := jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest")
	c.Assert(info.Args, gc.Equals, "<command> ...")
	c.Assert(info.Doc, gc.Matches, helpCommandsText)

	jc, _, err := initDefenestrate([]string{"discombobulate"})
	c.Assert(err, gc.ErrorMatches, "unrecognized command: jujutest discombobulate")
	info = jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest")
	c.Assert(info.Args, gc.Equals, "<command> ...")
	c.Assert(info.Doc, gc.Matches, "commands:\n    defenestrate  - defenestrate the juju"+docText+helpText)

	jc, tc, err := initDefenestrate([]string{"defenestrate"})
	c.Assert(err, gc.IsNil)
	c.Assert(tc.Option, gc.Equals, "")
	info = jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest defenestrate")
	c.Assert(info.Args, gc.Equals, "<something>")
	c.Assert(info.Doc, gc.Equals, "defenestrate-doc")

	_, tc, err = initDefenestrate([]string{"defenestrate", "--option", "firmly"})
	c.Assert(err, gc.IsNil)
	c.Assert(tc.Option, gc.Equals, "firmly")

	_, tc, err = initDefenestrate([]string{"defenestrate", "gibberish"})
	c.Assert(err, gc.ErrorMatches, `unrecognized args: \["gibberish"\]`)

	// --description must be used on it's own.
	_, _, err = initDefenestrate([]string{"--description", "defenestrate"})
	c.Assert(err, gc.ErrorMatches, `unrecognized args: \["defenestrate"\]`)

	// --no-alias is not a valid option if there is no alias file speciifed
	_, _, err = initDefenestrate([]string{"--no-alias", "defenestrate"})
	c.Assert(err, gc.ErrorMatches, `flag provided but not defined: --no-alias`)
}

func (s *SuperCommandSuite) TestUserAliasDispatch(c *gc.C) {
	// Can still use the full name.
	jc, tc, err := initDefenestrateWithAliases(c, []string{"defenestrate"})
	c.Assert(err, gc.IsNil)
	c.Assert(tc.Option, gc.Equals, "")
	info := jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest defenestrate")
	c.Assert(info.Args, gc.Equals, "<something>")
	c.Assert(info.Doc, gc.Equals, "defenestrate-doc")

	jc, tc, err = initDefenestrateWithAliases(c, []string{"def"})
	c.Assert(err, gc.IsNil)
	c.Assert(tc.Option, gc.Equals, "")
	info = jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest defenestrate")

	jc, tc, err = initDefenestrateWithAliases(c, []string{"be-firm"})
	c.Assert(err, gc.IsNil)
	c.Assert(tc.Option, gc.Equals, "firmly")
	info = jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest defenestrate")

	_, _, err = initDefenestrateWithAliases(c, []string{"--no-alias", "def"})
	c.Assert(err, gc.ErrorMatches, "unrecognized command: jujutest def")

	// Aliases to missing values are converted before lookup.
	_, _, err = initDefenestrateWithAliases(c, []string{"other"})
	c.Assert(err, gc.ErrorMatches, "unrecognized command: jujutest missing")
}

func (s *SuperCommandSuite) TestRegister(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{Name: "jujutest"})
	jc.Register(&TestCommand{Name: "flip"})
	jc.Register(&TestCommand{Name: "flap"})
	badCall := func() { jc.Register(&TestCommand{Name: "flap"}) }
	c.Assert(badCall, gc.PanicMatches, `command already registered: "flap"`)
}

func (s *SuperCommandSuite) TestAliasesRegistered(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{Name: "jujutest"})
	jc.Register(&TestCommand{Name: "flip", Aliases: []string{"flap", "flop"}})

	info := jc.Info()
	c.Assert(info.Doc, gc.Equals, `commands:
    documentation - Generate the documentation for all commands
    flap          - Alias for 'flip'.
    flip          - flip the juju
    flop          - Alias for 'flip'.
    help          - Show help on a command or other topic.`)
}

func (s *SuperCommandSuite) TestInfo(c *gc.C) {
	commandsDoc := `commands:
    documentation - Generate the documentation for all commands
    flapbabble    - flapbabble the juju
    flip          - flip the juju`

	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "jujutest",
		Purpose: "to be purposeful",
		Doc:     "doc\nblah\ndoc",
	})
	info := jc.Info()
	c.Assert(info.Name, gc.Equals, "jujutest")
	c.Assert(info.Purpose, gc.Equals, "to be purposeful")
	// info doc starts with the jc.Doc and ends with the help command
	c.Assert(info.Doc, gc.Matches, jc.Doc+"(.|\n)*")
	c.Assert(info.Doc, gc.Matches, "(.|\n)*"+helpCommandsText)

	jc.Register(&TestCommand{Name: "flip"})
	jc.Register(&TestCommand{Name: "flapbabble"})
	info = jc.Info()
	c.Assert(info.Doc, gc.Matches, jc.Doc+"\n\n"+commandsDoc+helpText)

	jc.Doc = ""
	info = jc.Info()
	c.Assert(info.Doc, gc.Matches, commandsDoc+helpText)
}

type testVersionFlagCommand struct {
	cmd.CommandBase
	version string
}

func (c *testVersionFlagCommand) Info() *cmd.Info {
	return &cmd.Info{Name: "test"}
}

func (c *testVersionFlagCommand) SetFlags(f *gnuflag.FlagSet) {
	f.StringVar(&c.version, "version", "", "")
}

func (c *testVersionFlagCommand) Run(_ *cmd.Context) error {
	return nil
}

func (s *SuperCommandSuite) TestVersionVerb(c *gc.C) {
	s.testVersion(c, []string{"version"})
}

func (s *SuperCommandSuite) TestVersionFlag(c *gc.C) {
	s.testVersion(c, []string{"--version"})
}

func (s *SuperCommandSuite) testVersion(c *gc.C, params []string) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "jujutest",
		Purpose: "to be purposeful",
		Doc:     "doc\nblah\ndoc",
		Version: "111.222.333",
	})
	testVersionFlagCommand := &testVersionFlagCommand{}
	jc.Register(testVersionFlagCommand)

	code := cmd.Main(jc, s.ctx, params)
	c.Check(code, gc.Equals, 0)
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "")
	c.Assert(cmdtesting.Stdout(s.ctx), gc.Equals, "111.222.333\n")
}

func (s *SuperCommandSuite) TestVersionFlagSpecific(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "jujutest",
		Purpose: "to be purposeful",
		Doc:     "doc\nblah\ndoc",
		Version: "111.222.333",
	})
	testVersionFlagCommand := &testVersionFlagCommand{}
	jc.Register(testVersionFlagCommand)

	// juju test --version should update testVersionFlagCommand.version,
	// and there should be no output. The --version flag on the 'test'
	// subcommand has a different type to the "juju --version" flag.
	code := cmd.Main(jc, s.ctx, []string{"test", "--version=abc.123"})
	c.Check(code, gc.Equals, 0)
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "")
	c.Assert(cmdtesting.Stdout(s.ctx), gc.Equals, "")
	c.Assert(testVersionFlagCommand.version, gc.Equals, "abc.123")
}

func (s *SuperCommandSuite) TestVersionNotProvidedVerb(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "jujutest",
		Purpose: "to be purposeful",
		Doc:     "doc\nblah\ndoc",
	})
	// juju version
	code := cmd.Main(jc, s.ctx, []string{"version"})
	c.Check(code, gc.Not(gc.Equals), 0)
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "ERROR unrecognized command: jujutest version\n")
}

func (s *SuperCommandSuite) TestVersionNotProvidedFlag(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "jujutest",
		Purpose: "to be purposeful",
		Doc:     "doc\nblah\ndoc",
	})
	// juju --version
	code := cmd.Main(jc, s.ctx, []string{"--version"})
	c.Check(code, gc.Not(gc.Equals), 0)
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "ERROR flag provided but not defined: --version\n")
}

func (s *SuperCommandSuite) TestVersionNotProvidedOption(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:    "jujutest",
		Purpose: "to be purposeful",
		Doc:     "doc\nblah\ndoc",
	})
	// juju --version where flags are known as options
	jc.FlagKnownAs = "option"
	code := cmd.Main(jc, s.ctx, []string{"--version"})
	c.Check(code, gc.Not(gc.Equals), 0)
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "ERROR option provided but not defined: --version\n")
}

func (s *SuperCommandSuite) TestLogging(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	sc.Register(&TestCommand{Name: "blah"})
	code := cmd.Main(sc, s.ctx, []string{"blah", "--option", "error", "--debug"})
	c.Assert(code, gc.Equals, 1)
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Matches, `(?m)ERROR BAM!\n.* DEBUG .* error stack: \n.*`)
}

type notifyTest struct {
	usagePrefix string
	name        string
	expectName  string
}

func (s *SuperCommandSuite) TestNotifyRunJujuJuju(c *gc.C) {
	s.testNotifyRun(c, notifyTest{"juju", "juju", "juju"})
}
func (s *SuperCommandSuite) TestNotifyRunSomethingElse(c *gc.C) {
	s.testNotifyRun(c, notifyTest{"something", "else", "something else"})
}
func (s *SuperCommandSuite) TestNotifyRunJuju(c *gc.C) {
	s.testNotifyRun(c, notifyTest{"", "juju", "juju"})
}
func (s *SuperCommandSuite) TestNotifyRunMyApp(c *gc.C) {
	s.testNotifyRun(c, notifyTest{"", "myapp", "myapp"})
}

func (s *SuperCommandSuite) testNotifyRun(c *gc.C, test notifyTest) {
	notifyName := ""
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: test.usagePrefix,
		Name:        test.name,
		NotifyRun: func(name string) {
			notifyName = name
		},
		Log: &cmd.Log{},
	})
	sc.Register(&TestCommand{Name: "blah"})
	code := cmd.Main(sc, s.ctx, []string{"blah", "--option", "error"})
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Matches, "ERROR BAM!\n")
	c.Assert(code, gc.Equals, 1)
	c.Assert(notifyName, gc.Equals, test.expectName)
}

func (s *SuperCommandSuite) TestDescription(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{Name: "jujutest", Purpose: "blow up the death star"})
	jc.Register(&TestCommand{Name: "blah"})
	code := cmd.Main(jc, s.ctx, []string{"blah", "--description"})
	c.Assert(code, gc.Equals, 0)
	c.Assert(cmdtesting.Stdout(s.ctx), gc.Equals, "blow up the death star\n")
}

func NewSuperWithCallback(callback func(*cmd.Context, string, []string) error) cmd.Command {
	return cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:            "jujutest",
		Log:             &cmd.Log{},
		MissingCallback: callback,
	})
}

func (s *SuperCommandSuite) TestMissingCallback(c *gc.C) {
	var calledName string
	var calledArgs []string

	callback := func(ctx *cmd.Context, subcommand string, args []string) error {
		calledName = subcommand
		calledArgs = args
		return nil
	}

	code := cmd.Main(
		NewSuperWithCallback(callback),
		cmdtesting.Context(c),
		[]string{"foo", "bar", "baz", "--debug"})
	c.Assert(code, gc.Equals, 0)
	c.Assert(calledName, gc.Equals, "foo")
	c.Assert(calledArgs, gc.DeepEquals, []string{"bar", "baz", "--debug"})
}

func (s *SuperCommandSuite) TestMissingCallbackErrors(c *gc.C) {
	callback := func(ctx *cmd.Context, subcommand string, args []string) error {
		return fmt.Errorf("command not found %q", subcommand)
	}

	code := cmd.Main(NewSuperWithCallback(callback), s.ctx, []string{"foo"})
	c.Assert(code, gc.Equals, 1)
	c.Assert(cmdtesting.Stdout(s.ctx), gc.Equals, "")
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "ERROR command not found \"foo\"\n")
}

func (s *SuperCommandSuite) TestMissingCallbackContextWiredIn(c *gc.C) {
	callback := func(ctx *cmd.Context, subcommand string, args []string) error {
		fmt.Fprintf(ctx.Stdout, "this is std out")
		fmt.Fprintf(ctx.Stderr, "this is std err")
		return nil
	}

	code := cmd.Main(NewSuperWithCallback(callback), s.ctx, []string{"foo", "bar", "baz", "--debug"})
	c.Assert(code, gc.Equals, 0)
	c.Assert(cmdtesting.Stdout(s.ctx), gc.Equals, "this is std out")
	c.Assert(cmdtesting.Stderr(s.ctx), gc.Equals, "this is std err")
}

func (s *SuperCommandSuite) TestSupercommandAliases(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:        "jujutest",
		UsagePrefix: "juju",
	})
	sub := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:        "jubar",
		UsagePrefix: "juju jujutest",
		Aliases:     []string{"jubaz", "jubing"},
	})
	info := sub.Info()
	c.Check(info.Aliases, gc.DeepEquals, []string{"jubaz", "jubing"})
	jc.Register(sub)
	for _, name := range []string{"jubar", "jubaz", "jubing"} {
		c.Logf("testing command name %q", name)
		s.SetUpTest(c)
		code := cmd.Main(jc, s.ctx, []string{name, "--help"})
		c.Assert(code, gc.Equals, 0)
		c.Assert(cmdtesting.Stdout(s.ctx), gc.Matches, "(?s).*Usage: juju jujutest jubar.*")
		c.Assert(cmdtesting.Stdout(s.ctx), gc.Matches, "(?s).*Aliases: jubaz, jubing.*")
		s.TearDownTest(c)
	}
}

type simple struct {
	cmd.CommandBase
	name string
	args []string
}

var _ cmd.Command = (*simple)(nil)

func (s *simple) Info() *cmd.Info {
	return &cmd.Info{Name: s.name, Purpose: "to be simple"}
}

func (s *simple) Init(args []string) error {
	s.args = args
	return nil
}

func (s *simple) Run(ctx *cmd.Context) error {
	fmt.Fprintf(ctx.Stdout, "%s %s\n", s.name, strings.Join(s.args, ", "))
	return nil
}

type deprecate struct {
	replacement string
	obsolete    bool
}

func (d deprecate) Deprecated() (bool, string) {
	if d.replacement == "" {
		return false, ""
	}
	return true, d.replacement
}
func (d deprecate) Obsolete() bool {
	return d.obsolete
}

func (s *SuperCommandSuite) TestRegisterAlias(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name: "jujutest",
	})
	jc.Register(&simple{name: "test"})
	jc.RegisterAlias("foo", "test", nil)
	jc.RegisterAlias("bar", "test", deprecate{replacement: "test"})
	jc.RegisterAlias("baz", "test", deprecate{obsolete: true})

	c.Assert(
		func() { jc.RegisterAlias("omg", "unknown", nil) },
		gc.PanicMatches, `"unknown" not found when registering alias`)

	info := jc.Info()
	// NOTE: deprecated `bar` not shown in commands.
	c.Assert(info.Doc, gc.Equals, `commands:
    documentation - Generate the documentation for all commands
    foo           - Alias for 'test'.
    help          - Show help on a command or other topic.
    test          - to be simple`)

	for _, test := range []struct {
		name   string
		stdout string
		stderr string
		code   int
	}{
		{
			name:   "test",
			stdout: "test arg\n",
		}, {
			name:   "foo",
			stdout: "test arg\n",
		}, {
			name:   "bar",
			stdout: "test arg\n",
			stderr: "WARNING \"bar\" is deprecated, please use \"test\"\n",
		}, {
			name:   "baz",
			stderr: "ERROR unrecognized command: jujutest baz\n",
			code:   2,
		},
	} {
		s.SetUpTest(c)
		code := cmd.Main(jc, s.ctx, []string{test.name, "arg"})
		c.Check(code, gc.Equals, test.code)
		c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, test.stdout)
		c.Check(cmdtesting.Stderr(s.ctx), gc.Equals, test.stderr)
		s.TearDownTest(c)
	}
}

func (s *SuperCommandSuite) TestRegisterSuperAlias(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name: "jujutest",
	})
	jc.Register(&simple{name: "test"})
	sub := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name:        "bar",
		UsagePrefix: "jujutest",
		Purpose:     "bar functions",
	})
	jc.Register(sub)
	sub.Register(&simple{name: "foo"})

	c.Assert(
		func() { jc.RegisterSuperAlias("bar-foo", "unknown", "foo", nil) },
		gc.PanicMatches, `"unknown" not found when registering alias`)
	c.Assert(
		func() { jc.RegisterSuperAlias("bar-foo", "test", "foo", nil) },
		gc.PanicMatches, `"test" is not a SuperCommand`)
	c.Assert(
		func() { jc.RegisterSuperAlias("bar-foo", "bar", "unknown", nil) },
		gc.PanicMatches, `"unknown" not found as a command in "bar"`)

	jc.RegisterSuperAlias("bar-foo", "bar", "foo", nil)
	jc.RegisterSuperAlias("bar-dep", "bar", "foo", deprecate{replacement: "bar foo"})
	jc.RegisterSuperAlias("bar-ob", "bar", "foo", deprecate{obsolete: true})

	info := jc.Info()
	// NOTE: deprecated `bar` not shown in commands.
	c.Assert(info.Doc, gc.Equals, `commands:
    bar           - bar functions
    bar-foo       - Alias for 'bar foo'.
    documentation - Generate the documentation for all commands
    help          - Show help on a command or other topic.
    test          - to be simple`)

	for _, test := range []struct {
		args   []string
		stdout string
		stderr string
		code   int
	}{
		{
			args:   []string{"bar", "foo", "arg"},
			stdout: "foo arg\n",
		}, {
			args:   []string{"bar-foo", "arg"},
			stdout: "foo arg\n",
		}, {
			args:   []string{"bar-dep", "arg"},
			stdout: "foo arg\n",
			stderr: "WARNING \"bar-dep\" is deprecated, please use \"bar foo\"\n",
		}, {
			args:   []string{"bar-ob", "arg"},
			stderr: "ERROR unrecognized command: jujutest bar-ob\n",
			code:   2,
		},
	} {
		s.SetUpTest(c)
		code := cmd.Main(jc, s.ctx, test.args)
		c.Check(code, gc.Equals, test.code)
		c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, test.stdout)
		c.Check(cmdtesting.Stderr(s.ctx), gc.Equals, test.stderr)
		s.TearDownTest(c)
	}
}

type simpleAlias struct {
	simple
}

func (s *simpleAlias) Info() *cmd.Info {
	return &cmd.Info{Name: s.name, Purpose: "to be simple with an alias",
		Aliases: []string{s.name + "-alias"}}
}

func (s *SuperCommandSuite) TestRegisterDeprecated(c *gc.C) {
	jc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		Name: "jujutest",
	})

	// Test that calling with a nil command will not panic
	jc.RegisterDeprecated(nil, nil)

	jc.RegisterDeprecated(&simpleAlias{simple{name: "test-non-dep"}}, nil)
	jc.RegisterDeprecated(&simpleAlias{simple{name: "test-dep"}}, deprecate{replacement: "test-dep-new"})
	jc.RegisterDeprecated(&simpleAlias{simple{name: "test-ob"}}, deprecate{obsolete: true})

	badCall := func() {
		jc.RegisterDeprecated(&simpleAlias{simple{name: "test-dep"}}, deprecate{replacement: "test-dep-new"})
	}
	c.Assert(badCall, gc.PanicMatches, `command already registered: "test-dep"`)

	for _, test := range []struct {
		args   []string
		stdout string
		stderr string
		code   int
	}{
		{
			args:   []string{"test-non-dep", "arg"},
			stdout: "test-non-dep arg\n",
		}, {
			args:   []string{"test-non-dep-alias", "arg"},
			stdout: "test-non-dep arg\n",
		}, {
			args:   []string{"test-dep", "arg"},
			stdout: "test-dep arg\n",
			stderr: "WARNING \"test-dep\" is deprecated, please use \"test-dep-new\"\n",
		}, {
			args:   []string{"test-dep-alias", "arg"},
			stdout: "test-dep arg\n",
			stderr: "WARNING \"test-dep-alias\" is deprecated, please use \"test-dep-new\"\n",
		}, {
			args:   []string{"test-ob", "arg"},
			stderr: "ERROR unrecognized command: jujutest test-ob\n",
			code:   2,
		}, {
			args:   []string{"test-ob-alias", "arg"},
			stderr: "ERROR unrecognized command: jujutest test-ob-alias\n",
			code:   2,
		},
	} {
		s.SetUpTest(c)
		code := cmd.Main(jc, s.ctx, test.args)
		c.Check(code, gc.Equals, test.code)
		c.Check(cmdtesting.Stderr(s.ctx), gc.Equals, test.stderr)
		c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, test.stdout)
		s.TearDownTest(c)
	}
}

func (s *SuperCommandSuite) TestGlobalFlagsBeforeCommand(c *gc.C) {
	flag := ""
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		GlobalFlags: flagAdderFunc(func(fset *gnuflag.FlagSet) {
			fset.StringVar(&flag, "testflag", "", "global test flag")
		}),
		Log: &cmd.Log{},
	})
	sc.Register(&TestCommand{Name: "blah"})
	code := cmd.Main(sc, s.ctx, []string{
		"--testflag=something",
		"blah",
		"--option=testoption",
	})
	c.Assert(code, gc.Equals, 0)
	c.Assert(flag, gc.Equals, "something")
	c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, "testoption\n")
}

func (s *SuperCommandSuite) TestGlobalFlagsAfterCommand(c *gc.C) {
	flag := ""
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		GlobalFlags: flagAdderFunc(func(fset *gnuflag.FlagSet) {
			fset.StringVar(&flag, "testflag", "", "global test flag")
		}),
		Log: &cmd.Log{},
	})
	sc.Register(&TestCommand{Name: "blah"})
	code := cmd.Main(sc, s.ctx, []string{
		"blah",
		"--option=testoption",
		"--testflag=something",
	})
	c.Assert(code, gc.Equals, 0)
	c.Assert(flag, gc.Equals, "something")
	c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, "testoption\n")
}

func (s *SuperCommandSuite) TestSuperSetFlags(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
		FlagKnownAs: "option",
	})
	s.assertFlagsAlias(c, sc, "option")
}

func (s *SuperCommandSuite) TestSuperSetFlagsDefault(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	s.assertFlagsAlias(c, sc, "flag")
}

func (s *SuperCommandSuite) assertFlagsAlias(c *gc.C, sc *cmd.SuperCommand, expectedAlias string) {
	sc.Register(&TestCommand{Name: "blah"})
	code := cmd.Main(sc, s.ctx, []string{
		"blah",
		"--fluffs",
	})
	c.Assert(code, gc.Equals, 2)
	c.Check(s.ctx.IsSerial(), gc.Equals, false)
	c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, "")
	c.Check(cmdtesting.Stderr(s.ctx), gc.Equals, fmt.Sprintf("ERROR %v provided but not defined: --fluffs\n", expectedAlias))
}

func (s *SuperCommandSuite) TestErrInJson(c *gc.C) {
	output := cmd.Output{}
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
		GlobalFlags: flagAdderFunc(func(fset *gnuflag.FlagSet) {
			output.AddFlags(fset, "json", map[string]cmd.Formatter{"json": cmd.FormatJson})
		}),
	})
	s.assertFormattingErr(c, sc, "json")
}

func (s *SuperCommandSuite) TestErrInYaml(c *gc.C) {
	output := cmd.Output{}
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
		GlobalFlags: flagAdderFunc(func(fset *gnuflag.FlagSet) {
			output.AddFlags(fset, "yaml", map[string]cmd.Formatter{"yaml": cmd.FormatYaml})
		}),
	})
	s.assertFormattingErr(c, sc, "yaml")
}

func (s *SuperCommandSuite) TestErrInJsonWithOutput(c *gc.C) {
	output := cmd.Output{}
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
		GlobalFlags: flagAdderFunc(func(fset *gnuflag.FlagSet) {
			output.AddFlags(fset, "json", map[string]cmd.Formatter{"json": cmd.FormatJson})
		}),
	})
	// This command will throw an error during the run after logging a structured output.
	testCmd := &TestCommand{
		Name:   "blah",
		Option: "error",
		CustomRun: func(ctx *cmd.Context) error {
			output.Write(ctx, struct {
				Name string `json:"name"`
			}{Name: "test"})
			return errors.New("BAM!")
		},
	}
	sc.Register(testCmd)
	code := cmd.Main(sc, s.ctx, []string{
		"blah",
		"--format=json",
		"--option=error",
	})
	c.Assert(code, gc.Equals, 1)
	c.Check(s.ctx.IsSerial(), gc.Equals, true)
	c.Check(cmdtesting.Stderr(s.ctx), gc.Matches, "ERROR BAM!\n")
	c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, "{\"name\":\"test\"}\n")
}

func (s *SuperCommandSuite) assertFormattingErr(c *gc.C, sc *cmd.SuperCommand, format string) {
	// This command will throw an error during the run
	testCmd := &TestCommand{Name: "blah", Option: "error"}
	sc.Register(testCmd)
	formatting := fmt.Sprintf("--format=%v", format)
	code := cmd.Main(sc, s.ctx, []string{
		"blah",
		formatting,
		"--option=error",
	})
	c.Assert(code, gc.Equals, 1)
	c.Check(s.ctx.IsSerial(), gc.Equals, true)
	c.Check(cmdtesting.Stderr(s.ctx), gc.Matches, "ERROR BAM!\n")
	c.Check(cmdtesting.Stdout(s.ctx), gc.Equals, "{}\n")
}

type flagAdderFunc func(*gnuflag.FlagSet)

func (f flagAdderFunc) AddFlags(fset *gnuflag.FlagSet) {
	f(fset)
}

func (s *SuperCommandSuite) TestFindClosestSubCommand(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	name, _, ok := sc.FindClosestSubCommand("halp")
	c.Assert(ok, gc.Equals, true)
	c.Assert(name, gc.Equals, "help")
}

func (s *SuperCommandSuite) TestFindClosestSubCommandReturnsExactMatch(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	name, _, ok := sc.FindClosestSubCommand("help")
	c.Assert(ok, gc.Equals, true)
	c.Assert(name, gc.Equals, "help")
}

func (s *SuperCommandSuite) TestFindClosestSubCommandReturnsNonExactMatch(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	_, _, ok := sc.FindClosestSubCommand("sillycommand")
	c.Assert(ok, gc.Equals, false)
}

func (s *SuperCommandSuite) TestFindClosestSubCommandReturnsWithPartialName(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	name, _, ok := sc.FindClosestSubCommand("hel")
	c.Assert(ok, gc.Equals, true)
	c.Assert(name, gc.Equals, "help")
}

func (s *SuperCommandSuite) TestFindClosestSubCommandReturnsWithLessMisspeltName(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	name, _, ok := sc.FindClosestSubCommand("hlp")
	c.Assert(ok, gc.Equals, true)
	c.Assert(name, gc.Equals, "help")
}

func (s *SuperCommandSuite) TestFindClosestSubCommandReturnsWithMoreName(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	name, _, ok := sc.FindClosestSubCommand("helper")
	c.Assert(ok, gc.Equals, true)
	c.Assert(name, gc.Equals, "help")
}

func (s *SuperCommandSuite) TestFindClosestSubCommandReturnsConsistentResults(c *gc.C) {
	sc := cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "juju",
		Name:        "command",
		Log:         &cmd.Log{},
	})
	sc.Register(cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "hxlp",
		Name:        "hxlp",
		Log:         &cmd.Log{},
	}))
	sc.Register(cmd.NewSuperCommand(cmd.SuperCommandParams{
		UsagePrefix: "hflp",
		Name:        "hflp",
		Log:         &cmd.Log{},
	}))
	name, _, ok := sc.FindClosestSubCommand("helper")
	c.Assert(ok, gc.Equals, true)
	c.Assert(name, gc.Equals, "help")
}
