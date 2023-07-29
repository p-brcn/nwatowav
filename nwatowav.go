package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hasenbanck/nwa"
)

var inputfile = flag.String("inputfile", "", "path to the input file.")
var inputdir = flag.String("inputdir", "", "path to the input directory containing *.nwa files.")

type fileType int

const (
	NONE fileType = iota
	NWA
	NWK
	OVK
)

func main() {
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	// Check if a file or folder was dropped onto the executable.
	// If so, use it as inputfile or inputdir, if not provided via flags.
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if *inputfile == "" && isNWAFile(arg) {
			*inputfile = arg
		} else if *inputdir == "" && isDirectory(arg) {
			*inputdir = arg
		}
	}

	// Process the input file or directory based on flags or drag-and-drop.
	if *inputfile != "" {
		processNWAFile(*inputfile)
	}

	if *inputdir != "" {
		// Get a list of files with the ".nwa" extension in the input directory
		fileList, err := filepath.Glob(filepath.Join(*inputdir, "*.nwa"))
		if err != nil {
			log.Fatal(err)
		}

		if len(fileList) == 0 {
			log.Fatal("No *.nwa files found in the input directory.")
		}

		for _, inputFile := range fileList {
			processNWAFile(inputFile)
		}
	}
}

func processNWAFile(inputfile string) {
	var outfilename, outext, outpath string
	var filetype fileType
	var headblksz int64

	switch {
	case strings.Contains(inputfile, ".nwa"):
		filetype = NWA
		outext = "wav"
	case strings.Contains(inputfile, ".nwk"):
		filetype = NWK
		headblksz = 12
		outext = "wav"
	case strings.Contains(inputfile, ".ovk"):
		filetype = OVK
		headblksz = 16
		outext = "ogg"
	}
	if filetype == NONE {
		log.Printf("Skipping %s: This program can only handle .nwa/.nwk/.ovk files right now.", inputfile)
		return
	}

	outfilename = strings.Split(filepath.Base(inputfile), ".")[0]

	if filetype == NWA {
		file, err := os.Open(inputfile)
		defer file.Close()
		if err != nil {
			log.Fatal(err)
		}

		var data io.Reader
		if data, err = nwa.NewNwaFile(file); err != nil {
			log.Fatal(err)
		}

		outdir := filepath.Dir(inputfile)
		outpath := filepath.Join(outdir, fmt.Sprintf("%s.%s", outfilename, outext))

		out, err := os.Create(outpath)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()

		if _, err = io.Copy(out, data); err != nil {
			log.Fatal(err)
		}
	} else { // NWK or OVK files
		file, err := os.Open(inputfile)
		defer file.Close()
		if err != nil {
			log.Fatal(err)
		}

		var indexcount int32
		binary.Read(file, binary.LittleEndian, &indexcount)
		if indexcount <= 0 {
			if filetype == OVK {
				log.Printf("Skipping %s: Invalid Ogg-ovk file: index = %d\n", inputfile, indexcount)
			} else {
				log.Printf("Skipping %s: Invalid Koe-nkw file: index = %d\n", inputfile, indexcount)
			}
			return
		}

		tblsiz := make([]int32, indexcount)
		tbloff := make([]int32, indexcount)
		tblcnt := make([]int32, indexcount)
		tblorigsiz := make([]int32, indexcount)

		var i int32
		for i = 0; i < indexcount; i++ {
			buffer := new(bytes.Buffer)
			if count, err := io.CopyN(buffer, file, headblksz); count != headblksz || err != nil {
				log.Fatal("Couldn't read the index entries!")
			}
			binary.Read(buffer, binary.LittleEndian, &tblsiz[i])
			binary.Read(buffer, binary.LittleEndian, &tbloff[i])
			binary.Read(buffer, binary.LittleEndian, &tblcnt[i])
			binary.Read(buffer, binary.LittleEndian, &tblorigsiz[i])
		}

		c := make(chan int, indexcount)
		for i = 0; i < indexcount; i++ {
			if tbloff[i] <= 0 || tblsiz[i] <= 0 {
				log.Printf("Skipping %s: Invalid table[%d]: cnt %d, off %d, size %d\n", inputfile, i, tblcnt[i], tbloff[i], tblsiz[i])
				continue
			}
			outpath = filepath.Join(*inputdir, fmt.Sprintf("%s-%d.%s", outfilename, tblcnt[i], outext))
			go doDecode(filetype, outpath, inputfile, tbloff[i], tblsiz[i], c)
		}
		for i = 0; i < indexcount; i++ {
			<-c
		}
	}
}

func doDecode(filetype fileType, filename string, datafile string, offset int32, size int32, c chan int) {
	var count int64
	var data io.Reader

	file, err := os.Open(datafile)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	buffer := new(bytes.Buffer)
	file.Seek(int64(offset), 0)
	if count, err = io.CopyN(buffer, file, int64(size)); count != int64(size) || err != nil {
		log.Fatalf("Couldn't read the data for filename %s: off %d, size %d. Error: %s\n", filename, offset, size, err)
	}

	if filetype == NWK {
		if data, err = nwa.NewNwaFile(buffer); err != nil {
			log.Fatal(err)
		}
	} else {
		data = buffer
	}

	out, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, data); err != nil {
		log.Fatal(err)
	}
	c <- 1
}

func isNWAFile(file string) bool {
	return strings.HasSuffix(strings.ToLower(file), ".nwa")
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
