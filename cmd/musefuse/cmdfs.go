package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/davecgh/go-spew/spew"
	"github.com/dhowden/tag"
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/args"
	"github.com/shabbyrobe/cmdy/flags"
	service "github.com/shabbyrobe/go-service"
	"github.com/shabbyrobe/go-service/services"
	"github.com/shabbyrobe/musefuse"
	"github.com/shabbyrobe/musefuse/playlist"
)

type fsCommand struct {
	paths flags.StringList
	mount string
	web   string
	name  string
}

func (cmd *fsCommand) Synopsis() string { return "FS" }

func (cmd *fsCommand) Args() *args.ArgSet {
	set := args.NewArgSet()
	return set
}

func (cmd *fsCommand) Flags() *cmdy.FlagSet {
	set := cmdy.NewFlagSet()
	set.Var(&cmd.paths, "path", "Paths to scour (can pass multiple times)")
	set.StringVar(&cmd.mount, "mount", "", "Mount point")
	set.StringVar(&cmd.name, "name", "MuseFUSE", "Name")
	set.StringVar(&cmd.web, "web", "localhost:60608", "42. Web server, lets you browse the metadata..")
	return set
}

func (cmd *fsCommand) startWeb(ctx cmdy.Context, fs *musefuse.FS) error {
	ws := musefuse.NewWebServer(cmd.web, fs)
	return services.Start(ctx, service.New("", ws))
}

func (cmd *fsCommand) Run(ctx cmdy.Context) error {
	if len(cmd.paths) == 0 {
		return fmt.Errorf("musefuse: no -path supplied")
	}

	if cmd.mount == "" {
		return fmt.Errorf("musefuse: -mount is required")
	}

	lister := musefuse.NewLister(cmd.paths, musefuse.AudioExtensions, musefuse.PlaylistExtensions)

	files, err := lister.List(nil)
	if err != nil {
		return err
	}

	museFS := musefuse.NewFS()

	scratch := make([]byte, 65536)
	start := time.Now()
	for _, file := range files {
		if file.Kind != musefuse.FileAudio {
			continue
		}

		path := filepath.Join(file.Prefix, file.Path)

		tag, err := parseTag(path, scratch)
		entry := &musefuse.FileEntry{
			File:     file,
			Metadata: musefuse.MetadataFromTag(tag),
		}
		if err != nil {
			fmt.Printf("ERR %s %v\n", path, err)
			entry.Err = err.Error()
		}

		if err := museFS.AddAudio(entry); err != nil {
			fmt.Printf("ERR %s %v\n", path, err)
			entry.Err = err.Error() // FIXME: maybe a race?
		}
	}

	// Add playlists only after we have resolved all the files:
	for _, file := range files {
		if file.Kind != musefuse.FilePlaylist {
			continue
		}

		path := filepath.Join(file.Prefix, file.Path)

		playlist, err := playlist.LoadPlaylistFile(path)
		if err != nil {
			return err
		}

		spew.Dump(playlist.Files())
	}

	dur := time.Since(start)
	fmt.Println(dur, len(files), dur/time.Duration(len(files)))

	if cmd.web != "" {
		if err := cmd.startWeb(ctx, museFS); err != nil {
			return err
		}
	}

	// All this Ctrl-C stuff was a bit of a shitfight against the fuse
	// mount; it can be cleaned up quite a lot.
	stop := make(chan struct{}, 0)
	stopped := make(chan struct{}, 0)
	{
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt, os.Kill)
		go func() {
			<-sigc
			close(stop)
			wait := time.After(5 * time.Second)
			select {
			case <-stopped:
				os.Exit(1)
			case <-wait:
				log.Fatal("timed out waiting for shutdown")
			}
		}()
	}

	mountedFS, err := fuse.Mount(
		cmd.mount,
		fuse.FSName("musefuse"),
		fuse.Subtype("musefs"),
		fuse.LocalVolume(),
		fuse.VolumeName(cmd.name),
	)
	if err != nil {
		return err
	}
	defer close(stopped)
	defer mountedFS.Close()
	defer fuse.Unmount(cmd.mount)

	errc := make(chan error, 1)
	go func() {
		if err := fs.Serve(mountedFS, museFS); err != nil {
			errc <- err
		}
	}()

	select {
	case <-mountedFS.Ready:
	case err := <-errc:
		return err
	case <-stop:
		return fmt.Errorf("stopped")
	case <-ctx.Done():
		break
	}

	for {
		select {
		case err := <-errc:
			return err
		case <-stop:
			return fmt.Errorf("stopped")
		case <-ctx.Done():
			break
		}
	}

	if err := mountedFS.MountError; err != nil {
		return err
	}

	return nil
}

func parseTag(path string, scratch []byte) (tag.Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	meta, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
