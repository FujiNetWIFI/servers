# APOD for Fujinet

by Bill Kendrick, bill@newbreedsoftware.com, 2020-12-10 - 2021-06-05

## Purpose
Fetch [NASA's Astronomy Picture of the Day (APOD)](https://apod.nasa.gov/apod/),
convert it to a format suitable for quickly loading on an Atari, and make it
available via HTTP for an Atari with a #FujiNet and its `N:` device.

## Requirements
- A webserver
- [PHP](https://www.php.net) web scripting language
- [GNU Bash](https://www.gnu.org/software/bash/) shell
- [PHP's DOM library](https://www.php.net/manual/en/book.dom.php),
  for parsing the HTML and RSS feed XML from the APOD site
- [ImageMagick](https://imagemagick.org/), used for scaling and
  format conversion of the images fetched from the APOD site
- [Wget](https://www.gnu.org/software/wget/), used to fetch files
  off the web

People are encouraged to look at how this works and create improved
versions that are less "cobbled-together" than my original.

## How it works
The PHP script examines an image file on the server;
if it was from yesterday or before, the script fetches the HTML page
of the Astronomy Picture of the Day, looks for an "`<img src>`" tag,
and uses it to fetch an image (they are usually JPEG).

It then uses ImageMagick's `convert` to mangle the image down to a size
suitable for the Atari, and then feeds the resulting image into another
tiny PHP script that converts it to the proper bit depth.

It also grabs APOD's RSS feed to find the title and description of
the latest image and format it into a 159-character string -- 4 lines
of 40 columns, with whitespace padding, rather than line-breaks, for
simplest client-side implementation.

Finally, in either case (whether or not it had to fetch a new image),
the image file (7,680 bytes) and the title/description is spat out by
the script.  The Atari need simply make a request to the script on a
server, and display the results!

## Example code 1 - Atari BASIC
This Atari BASIC example loads an image in 80x192
16-greyshade mode (the default):

```
10 GRAPHICS 9:DIM A$(7):A$="hhh*LVd"
11 A$(4,4)=CHR$(170):REM Inverse *
12 A$(7,7)=CHR$(228):REM Inverse d
13 REM Machine language to get Atari OS to do block
14 REM data transfer
20 OPEN #1,4,0,"N:HTTP://SOMESERVER/apod/index.php"
30 POKE 852,PEEK(88):POKE 853,PEEK(89)
31 REM ICBAL/H (low/high address) for IOCB1; using low/high of
32 REM SAVMSC, the beginning of screen memory
40 POKE 856,0:POKE 857,30:REM 40 bytes per row x 192 rows,
41 REM aka 7,680 bytes, aka 30 256-byte pages
50 POKE 850,7:REM ICCOM for IOCB1; the command: "GET bytes"
60 X=USR(ADR(A$),16):REM 16 for IOCB1
70 CLOSE #1
99 GOTO 99
```

(Uses a 'famous' machine language routine to "`JMP`" (jump)
to the CIOV vector ($E456). A great explanation can be found
in the [AtariAge forums](https://atariage.com/forums/topic/174633-help-needed-atari800-basic-loading-and-saving-binary-files-on-cassette/?do=findComment&comment=2171905),
and in many classic Atari books and magazine articles, and
web forum threads.)

## Example code 2 - TurboBASIC XL
And here's a TurboBASIC XL example that loads a monochrome image,
and shows the title and description in the text window at the bottom:

```
5 DIM A$(159)
10 GRAPHICS 8
20 POKE 709,15:POKE 710,0:POKE 752,1:POKE 82,0:POKE 83,39
30 OPEN #1,4,0,"N:HTTP://SOMESERVER/apod/index.php?mode=8"
40 BGET #1,DPEEK(88),7680
50 INPUT #1,A$:? CHR$(125);A$;
60 CLOSE #1
99 GOTO 99
```

## Example code 3 - cc65
The 'official' APOD #FujiNet client is written in C and
compiled with cc65.  It uses direct SIO calls to access the
N: device of #FujiNet (no drivers needed).

See the code in GitHub at
https://github.com/FujiNetWIFI/fujinet-apps/tree/master/apod

## Modes
The APOD server webapp can produce various image formats,
depending on the "mode" argument sent to the script
(e.g., "...?mode=15")

 * 8 - "GRAPHICS 8" -- 320x192, monochrome (black & white)
 * 15 - "GRAPHICS 15" (aka "7+") -- 160x192, 4 (best) colors
 * 15dli - same as above, but with 4 best colors _per scanline_
 * 9 - "GRAPHICS 9" -- 80x192, 16 greys
 * rgb9 -- "ColorView 9" -- 80x192, 4096 colors (via 16 shades * 3 (red, green, and blue))

## Dealing with Non-imagery
Sometimes the APOD is actually a video.  If an embedded YouTube clip
is detected (within an `<iframe>` on the APOD web page), we'll fetch and
convert the default thumbnail image for the video.

## Sample images
A number of local sample images are available; add a
"sample" argument to the URL (e.g., "...?mode=8&sample=2")
to receive one of them (after converting, if necessary; these
can be used during development to help fine-tune the conversion
process).

## Possible expansions
### More Atari Image Types
More software-driven modes like the two low resolution 256-color APAC
(Any Point, Any Color) modes (80x192 via flickering, or 80x96 static),
or other resolutions of ColorView mode (320x192 8 color and 160x192 64 color),
HIP, RIP, etc.

### Overall color hints
When sending monochrome images, provide a hint as to
what hue the image is, overall (e.g., if it's a big
mostly-purple nebula, why not show the image in shades of
purple?)

