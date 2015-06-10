# godd - a dd like tool with progress bar

A small dd like tool with progressbar. Useful when writing disk images to 
e.g. sd-cards.

It will also sanity check that you don't write to a mounted device
(no more accidental dd of your main hdd anymore).

## Usage

```
$ sudo godd ubuntu-15.04-snappy-armhf-bbb.img /dev/sdd
1.29 GB / 3.63 GB [================>-----------------------------] 35.56 % 3m28s
```
