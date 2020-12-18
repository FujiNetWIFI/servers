# APOD for Fujinet

WIP by Bill Kendrick, bill@newbreedsoftware.com, 2020-12-10 - 2020-12-12

## Purpose
Fetch [NASA's Astronomy Picture of the Day (APOD)](https://apod.nasa.gov/apod/),
convert it to a format suitable for quickly loading on an Atari, and make it
available via HTTP for an Atari with a #FujiNet and its `N:` device.

## How it works
The PHP script examines an image file on the server;
if it was from yesterday or before, the script fetches the HTML page
of the Astronomy Picture of the Day, looks for an "`<img src>`" tag,
and uses it to fetch an image (they are usually JPEG).

It then uses ImageMagick's `convert` to mangle the image down to a size
suitable for the Atari, and then feeds the resulting image into another
tiny PHP script that converts it to the proper bit depth.

Finally, in either case (whether or not it had to fetch a new image),
the image file is spat out by the script.  The Atari need simply
make a request to the script on a server, and display the results.

## Example code
This TurboBASIC XL example loads an image in 80x192
16-greyshade mode (the default):

```
10 GRAPHICS 9
20 OPEN #1,4,0,"N:HTTP://SOMESERVER/apod/index.php"
30 BGET #1,DPEEK(88),7680
40 CLOSE #1
50 GOTO 50
```

And this loads a monochrome image:
```
10 GRAPHICS 8+16:POKE 709,15:POKE 710,0
20 OPEN #1,4,0,"N:HTTP://SOMESERVER/apod/index.php?mode=8"
30 BGET #1,DPEEK(88),7680
40 CLOSE #1
50 GOTO 50
```

## Possible expansion
### More Atari Image Types
Software-driven modes like the low resolution 256-color APAC (Any Point,
Any Color) or various resolution 8-, 64- and 4096-color ColorView modes,
HIP, RIP, etc.

### Descriptions
Fetch the title and description of the APOD, send it in the
response, and display it on the Atari.

## Non-imagery
Sometimes the APOD is actually a video (e.g., an embedded YouTube clip).
We could get really clever, determine how to grab YouTube's preview image
for the video, and send that to the Atari (it's better than nothing!)
(See my bug to KDE's "Picture of the Day" wallpaper component, regarding
APOD videos: https://bugs.kde.org/show_bug.cgi?id=425058)

