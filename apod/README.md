# APOD for Fujinet

by Bill Kendrick, bill@newbreedsoftware.com, 2020-12-10 - 2020-12-18

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

## Possible expansion
### More Atari Image Types
Software-driven modes like the low resolution 256-color APAC (Any Point,
Any Color) or various resolution 8-, 64- and 4096-color ColorView modes,
HIP, RIP, etc.

## Non-imagery
Sometimes the APOD is actually a video (e.g., an embedded YouTube clip).
We could get really clever, determine how to grab YouTube's preview image
for the video, and send that to the Atari (it's better than nothing!)
(See my bug to KDE's "Picture of the Day" wallpaper component, regarding
APOD videos: https://bugs.kde.org/show_bug.cgi?id=425058)

