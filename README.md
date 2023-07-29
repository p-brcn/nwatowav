Fork of github.com/hasenbanck/nwatowav
Adds converting a whole directory

## REQUIREMENTS
Go 1.20

## INSTALLATION
```
git clone github.com/p-brcn/nwatowav

cd nwatowav

go build
```

## USAGE
You can drag and drop a file or directory to the executable and it should convert it.

```
single file:
./nwatowav.exe --inputfile="FILENAME"

directory:
./nwatowav.exe --inputdir="DIR"
```

## LICENSE
This programm is licensed under the GNU General Public License version 3
(or later). You can find a copy of the license in the GPL.txt.

It was written while looking at the nwatowav programm written
by Kazunori "jagarl" Ueno, which has a BSDish license.
