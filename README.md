# godd - a dd like tool with progress bar

A small dd like tool with progressbar. Useful when writing disk images to 
e.g. sd-cards.

It will also sanity check that you don't write to a mounted device
(no more accidental dd of your main hdd anymore). It uses udev to
detect possible targets and will write synchronously (no need to run
"sync" manually after the image was written).

## Usage

Simple usage:
```
$ sudo godd ubuntu-15.04-snappy-armhf-bbb.img /dev/sdd
1.29 GB / 3.63 GB [================>-----------------------------] 35.56 % 3m28s
```

Without target it will display detected removable devices:
```
$ sudo godd ubuntu-15.04-snappy-armhf-bbb.img
No target selected, detected the following removable device:
  /dev/sdb

failed to parse args: please select target device
```
