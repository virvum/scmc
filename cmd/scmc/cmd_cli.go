package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/virvum/scmc/pkg/mycloud"

	"github.com/c-bata/go-prompt"
	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

type CliOptions struct {
	Username string
	Password string
	Quiet    bool
}

var cliOptions CliOptions

var cmdCli = &cobra.Command{
	Use:   "cli",
	Short: "Command-line interface to myCloud (similar to the commonly known ftp/sftp commands).",
	Long: strings.TrimSpace(`
The "cli" command launches an interactive command-line interface, making it
possible to interact with myCloud directly from a shell, similar to the
commonly known ftp/sftp commands.

Local commands are prefixed with an "l", remote commands have no prefix (e.g.
"cd" vs. "lcd").

Commands can also be piped directly via stdin, in order to run a sequence of
commands like so:

	echo put local-file.txt | scmc cli

Or:

	cat <<EOF | scmc cli
	put local-file.txt
	get remote-file.txt
	# ...
	EOF
	`),
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			username string
			password string
		)

		// TODO move this shit to aux.go or somewhere else

		reader := bufio.NewReader(os.Stdin)

		if cmd.Flags().Changed("username") {
			username = cliOptions.Username
		} else if cfg.Username != "" {
			username = cfg.Username
		} else {
			fmt.Print("Swisscom myCloud username: ")
			u, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reader.ReadString: %v\n", err)
			}

			username = u
		}

		if cmd.Flags().Changed("password") {
			password = cliOptions.Password
		} else if cfg.Password != "" {
			password = cfg.Password
		} else {
			fmt.Print("Swisscom myCloud password: ")
			p, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("terminal.ReadPassword: %v\n", err)
			}

			password = string(p)
			fmt.Println()
		}

		cliOptions.Username = username
		cliOptions.Password = password

		return runCli()
	},
}

func init() {
	cmdRoot.AddCommand(cmdCli)

	f := cmdCli.Flags()
	f.StringVarP(&cliOptions.Username, "username", "u", os.Getenv("MYCLOUD_USERNAME"), "Swisscom myCloud username (default: $MYCLOUD_USERNAME)")
	f.StringVarP(&cliOptions.Password, "password", "p", os.Getenv("MYCLOUD_PASSWORD"), "Swisscom myCloud password (default: $MYCLOUD_PASSWORD)")
	f.BoolVarP(&cliOptions.Quiet, "quiet", "q", false, "suppress information output before the CLI spawns")
}

func uploadDir(p string) error {
	entries, err := ioutil.ReadDir(p)
	if err != nil {
		return fmt.Errorf("ioutil.ReadDir(%s): %v\n", p, err)
	}

	rp := path.Join(rpwd, p) + "/"

	fmt.Fprintf(os.Stderr, "creating remote directory '%s'\n", rp[1:])

	if err := mc.CreateDirectory(rp); err != nil {
		return fmt.Errorf("mc.CreateDirectory(%s): %v\n", p, err)
	}

	for _, f := range entries {
		switch mode := f.Mode(); {
		case mode.IsDir():
			dp := path.Join(p, f.Name())

			if err := uploadDir(dp); err != nil {
				return fmt.Errorf("uploadDir(%s): %v", dp, err)
			}
		case mode.IsRegular():
			fp := path.Join(p, f.Name())

			if err := uploadFile(fp); err != nil {
				return fmt.Errorf("uploadFile(%s): %v", fp, err)
			}
		default:
			return fmt.Errorf("invalid filetype: %v\n", path.Join(p, f.Name()))
		}
	}

	return nil
}

func uploadFile(p string) error {
	file, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("os.Open(%s): %v\n", p, err)
	}

	st, err := file.Stat()
	if err != nil {
		return fmt.Errorf("f.Stat: %v\n", err)
	}

	rp := path.Join(rpwd, path.Base(p))

	fmt.Fprintf(os.Stderr, "'%s' -> '%s'\n", p, rp)

	bar := pb.New64(st.Size())
	bar.SetRefreshRate(time.Second)
	bar.SetWriter(os.Stderr)
	// TODO maybe set a custom template: bar.SetTemplateString(...)

	reader := bar.NewProxyReader(file)

	bar.Start()

	if err := mc.CreateFile(rp, reader); err != nil {
		return fmt.Errorf("mc.CreateFile(%s): %v\n", rp, err)
	}

	bar.Finish()

	return nil
}

// upload uploads a single file or a directory and all of its contents.
func upload(p string) error {
	f, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("os.Stat(%s): %v\n", p, err)
	}

	switch mode := f.Mode(); {
	case mode.IsDir():
		return uploadDir(p)
	case mode.IsRegular():
		return uploadFile(p)
	}

	return fmt.Errorf("invalid filetype: %v\n", p)
}

func download(p string) error {
	metadata, err := mc.Metadata(p)
	if err != nil {
		return fmt.Errorf("mc.Metadata(%s): %v\n", p, err)
	}

	// TODO

	fmt.Printf("%v\n", metadata.Directories)
	fmt.Printf("%v\n", metadata.Files)
	return nil
}

func executor(s string) {
	if s = strings.TrimSpace(s); s != "" {
		cmd.SetArgs(strings.Fields(s))
		cmd.Execute()
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "help", Description: "Show available commands and additional information"},
		{Text: "pwd", Description: "Show remote working directory"},
		{Text: "lpwd", Description: "Show local working directory"},
		{Text: "cd", Description: "Change remote directory"},
		{Text: "lcd", Description: "Change local directory"},
		{Text: "ls", Description: "List remote files in current directory"},
		{Text: "lcd", Description: "List local files in current directory"},
		{Text: "put", Description: "Upload specified files or directories"},
		{Text: "get", Description: "Download specified files or directories"},
		{Text: "exit", Description: "Exit program"},
		{Text: "quit", Description: "Exit program"},
	}

	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func livePrefix() (string, bool) {
	return fmt.Sprintf("%s -> %s > ", lpwd, rpwd), true
}

var (
	mc   *mycloud.MyCloud
	lpwd string
	rpwd string         = "/"
	cmd  *cobra.Command = &cobra.Command{}
)

func runCli() error {
	cmd.AddCommand(&cobra.Command{
		Use:   "exit",
		Short: "exit the program",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(0)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "quit",
		Short: "exit the program",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(0)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "pwd",
		Short: "Show remote working directory",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(rpwd)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "lpwd",
		Short: "Show local working directory",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(lpwd)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cd [directory]",
		Short: "Change remote working directory to the given directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pwd := path.Clean(path.Join(rpwd, args[0]))

			if !strings.HasSuffix(pwd, "/") {
				pwd += "/"
			}

			// TODO instead of running mc.Metadata on the new pwd, run mc.Metadata on dirname(new pwd) and check whether the target directory is contained

			if _, err := mc.Metadata(pwd); err != nil {
				fmt.Fprintf(os.Stderr, "mc.Metadata: %v\n", err)
				return
			}

			rpwd = pwd
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "lcd [directory]",
		Short: "Change local working directory to the given directory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := os.Chdir(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "os.Chdir(%s): %v\n", args[0], err)
				return
			}

			pwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "os.Getwd: %v\n", err)
				return
			}

			lpwd = pwd
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List remote files in current directory",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			metadata, err := mc.Metadata(rpwd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mc.Metadata: %v", err)
			}

			// TODO sort by name after grouping dirs and files

			for _, d := range metadata.Directories {
				t := d.ModificationTime
				fmt.Printf("d %10s %d-%02d-%02d %02d:%02d:%02d %s\n", "-", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), d.Name)
			}

			for _, f := range metadata.Files {
				t := f.ModificationTime
				fmt.Printf("f %10d %d-%02d-%02d %02d:%02d:%02d %s\n", f.Length, t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), f.Name)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "lls",
		Short: "List local files in current directory",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			entries, err := ioutil.ReadDir(".")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ioutil.ReadDir: %v\n", err)
			}

			sort.Slice(entries, func(a, b int) bool {
				return entries[a].Mode().String()[0] == 'd'
			})

			// TODO sort by name after grouping dirs and files

			for _, f := range entries {
				t := f.ModTime()
				ft := f.Mode().String()[0]
				if ft == '-' {
					ft = 'f'
				}

				fmt.Printf("%c %10d %d-%02d-%02d %02d:%02d:%02d %s\n", ft, f.Size(), t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), f.Name())
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "put FILE [FILE ...]",
		Short: "Upload specified files or directories from the current local directory",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, p := range args {
				if err := upload(p); err != nil {
					fmt.Fprintf(os.Stderr, "upload(%s): %v", p, err)
					break
				}
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get FILE [FILE ...]",
		Short: "Download specified remote files or directories into the current local directory",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			for _, p := range args {
				if err := download(p); err != nil {
					fmt.Fprintf(os.Stderr, "download(%s): %v", p, err)
					break
				}
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cat FILE",
		Short: "Download remote file and output its content to the terminal",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			p := path.Join(rpwd, args[0])

			if err := mc.GetFile(p, os.Stdout, ""); err != nil {
				fmt.Fprintf(os.Stderr, "mc.GetFile(%s): %v\n", p, err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sha256 FILE [FILE ...]",
		Short: "Calculate SHA256 hash of a remote files",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			hasher := sha256.New()

			for _, fn := range args {
				p := path.Join(rpwd, fn)

				if err := mc.GetFile(p, hasher, ""); err != nil {
					fmt.Fprintf(os.Stderr, "mc.GetFile(%s): %v\n", p, err)
					break
				}

				fmt.Fprintf(os.Stderr, "%x %s\n", hasher.Sum(nil), fn)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "mkdir DIRECTORY [DIRECTORY ...]",
		Short: "Create directories and all parent directories",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var dirs []string

			for _, p := range args {
				dirs = append(dirs, path.Join(rpwd, p)+"/")
			}

			for _, dir := range dirs {
				if err := mc.CreateDirectory(dir); err != nil {
					fmt.Fprintf(os.Stderr, "mc.CreateDirectory(%s): %v\n", dir, err)
				} else {
					fmt.Fprintf(os.Stderr, "Directory '%s' created\n", dir)
				}
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "rm FILE [FILE ...]",
		Short: "Remove remote files",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var files []string

			for _, p := range args {
				files = append(files, path.Join(rpwd, p))
			}

			if err := mc.Delete(files); err != nil {
				fmt.Fprintf(os.Stderr, "mc.Delete(%s): %v\n", files, err)
			} else {
				for _, p := range files {
					fmt.Fprintf(os.Stderr, "File '%s' deleted\n", p)
				}
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "rmdir DIRECTORY [DIRECTORY ...]",
		Short: "Remove remote files",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var dirs []string

			for _, p := range args {
				dirs = append(dirs, path.Join(rpwd, p)+"/")
			}

			if err := mc.Delete(dirs); err != nil {
				fmt.Fprintf(os.Stderr, "mc.Delete(%s): %v\n", dirs, err)
			} else {
				for _, p := range dirs {
					fmt.Fprintf(os.Stderr, "Directory '%s' deleted\n", p)
				}
			}
		},
	})

	//cmd.AddCommand(&cobra.Command{
	//	Use:   "cksum LOCAL_FILE REMOTE_FILE",
	//	Short: "Compare a local file with a remote file by comparing their content",
	//	Args:  cobra.ExactArgs(2),
	//	Run: func(cmd *cobra.Command, args []string) {
	//		// TODO
	//	},
	//})

	cmd.AddCommand(&cobra.Command{
		Use:   "edit FILE",
		Short: "Edit a remote file using the editor specified in $EDITOR",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			p := path.Join(rpwd, args[0])

			file, err := ioutil.TempFile("/tmp", "scmc")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ioutil.TempFile: %v", err)
				return
			}

			defer func() {
				if err := os.Remove(file.Name()); err != nil {
					fmt.Fprintf(os.Stderr, "os.Remove: %v", err)
				}
			}()

			hasher := sha256.New()
			mw := io.MultiWriter(hasher, file)

			if err := mc.GetFile(p, mw, ""); err != nil {
				fmt.Fprintf(os.Stderr, "mc.GetFile(%s): %v\n", p, err)
				return
			}

			//original_hash := hasher.Sum(nil)

			editor, ok := os.LookupEnv("EDITOR")
			if !ok {
				editor = "vi"
			}

			c := exec.Command(editor, file.Name())
			c.Stdout = os.Stdout
			c.Stdin = os.Stdin
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "c.Run: %v\n", err)
				return
			}

			// TODO
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "ledit FILE",
		Short: "Edit a local file using the editor specified in $EDITOR",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			editor, ok := os.LookupEnv("EDITOR")
			if !ok {
				editor = "vi"
			}

			c := exec.Command(editor, path.Join(lpwd, args[0]))
			c.Stdout = os.Stdout
			c.Stdin = os.Stdin
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "c.Run: %v\n", err)
				return
			}
		},
	})

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	lpwd = pwd

	mc, err = mycloud.New(cliOptions.Username, cliOptions.Password, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mycloud.New: %v\n", err)
		os.Exit(1)
	}

	if terminal.IsTerminal(int(os.Stdin.Fd())) && terminal.IsTerminal(int(os.Stdout.Fd())) {
		if !cliOptions.Quiet {
			id, err := mc.Identity()
			if err != nil {
				fmt.Fprintf(os.Stderr, "mc.Identity: %v\n", err)
			} else {
				fmt.Printf("Logged in as %s (%s %s)\n", id.UserName, id.FirstName, id.LastName)
				fmt.Printf("Subscription: %s\n", id.Subscription.Name)
			}

			usage, err := mc.Usage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "mc.Usage: %v\n", err)
			} else {
				fmt.Printf("Usage: %s\n", bytesToSize(usage.TotalBytes))
			}
		}

		p := prompt.New(executor, completer, prompt.OptionLivePrefix(livePrefix), prompt.OptionTitle("scmc cli"))

		p.Run()
	} else {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			executor(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "scanner.Scan: %v\n", err)
		}
	}

	// TODO defer clear title
	// TODO only use autocomplete on TAB

	return nil
}
