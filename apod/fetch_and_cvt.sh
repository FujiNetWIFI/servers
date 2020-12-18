#!/bin/bash

rm "$3"
wget "$1" -O - | ./image_to_gr$2.sh > "$3"
chmod 666 "$3"

