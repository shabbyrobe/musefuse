MuseFUSE: Browse your music file's tags using the filesystem
============================================================

This was a fun little Sunday hack to make a read-only FUSE filesystem for
exploring tagged music files.

Basically I made it because I have a rather large mess on my hands here and
wanted to explore the data before I started to autotag everything, plus I
wanted to see how hard it was to use FUSE in Go (answer: not very, though the
docs are a bit pants).

Unfortunately, now I have two messes - this prototype has been really useful,
so now I have to clean up the code too!

To install:

    go install -u github.com/shabbyrobe/musefuse/cmd/musefuse

I've been using a mount point in `/media/$USER` because it shows up in
Nautilus, YMMV:

    sudo mkdir "/media/$USER/muse"
    sudo chown "$USER:$USER" "/media/$USER/muse"

Now mount the thing:

    musefuse fs -mount "/media/$USER/muse" -path ~/music/

Now you can explore!

    $ ls /media/bl/muse
    artist  artistalbum  failed  genre  unsorted  year

    $ ls /media/bl/muse/year
    1984  1987  1990  1993  1996  1999  2002  2005  2008  2011  2014  2017

It's a bit easier than I'd like at the moment for the program to shut down
before closing the mount. If you have a stray mount on your hands, just
`umount` it and you're good to go:

    sudo umount "/media/$USER/muse"

Here's what I plan to add:

- Finish the playlist handling
- Config files
- Multiple databases with configurable file extensions
- Make the webserver a bit better for exploring the metadata
- Re-scan databases periodically
- Trigger re-scan
- "Complete albums" - when tags have Tracks set, and all tracks are found.

Here's what I won't add:

- Something like `fsnotify` or `watcher`; I tried it and it was way too
  janky and I lost a lot of time that could've been better spent on the
  other list.
