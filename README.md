[![Build Status][travis-image]][travis-url]
# godd - a dd like tool with progress bar and minimal guard-rails

A small dd like tool with progressbar. Useful when writing disk images to 
e.g. SD cards.

It can autodetect (or be told) the compression format of the input file.

It will also sanity check that you don't write to a mounted device
(no more accidental dd of your main hdd anymore). It uses udev to
detect possible targets and will write synchronously (no need to run
"sync" manually after the image was written).

## Usage

Simple usage:
```
$ sudo godd ubuntu-15.04-snappy-armhf-bbb.img.xz /dev/sdd
1.29 GB / ? [-----------=-----------------------------]
```

Without target it will display detected removable devices:
```
$ sudo godd ubuntu-15.04-snappy-armhf-bbb.img
No target selected, detected the following removable device:
  /dev/sdb

failed to parse args: please select target device
```

[travis-image]: https://travis-ci.org/mvo5/godd.svg?branch=master
[travis-url]: https://travis-ci.org/mvo5/godd.svg?branch=master
