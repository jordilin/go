package draw

import (
	"fmt"
	"image"
	"os"
)

func stringnwidth(f *Font, s string, b []byte, r []rune) int {
	const Max = 64
	cbuf := make([]uint16, Max)
	var in input
	in.init(s, b, r)
	twid := 0
	for {
		max := Max
		n := 0
		var sf *Subfont
		var l, wid int
		var subfontname string
		for {
			if l, wid, subfontname = cachechars(f, &in, cbuf, max); l > 0 {
				break
			}
			if n++; n > 10 {
				r := in.ch
				name := f.Name
				if name == "" {
					name = "unnamed font"
				}
				sf.Free()
				fmt.Fprintf(os.Stderr, "stringwidth: bad character set for rune %U in %s\n", r, name)
				return twid
			}
			if subfontname != "" {
				sf.Free()
				var err error
				sf, err = getsubfont(f.Display, subfontname)
				if err != nil {
					if f.Display != nil && f != f.Display.DefaultFont {
						f = f.Display.DefaultFont
						continue
					}
					break
				}
				/*
				 * must not free sf until cachechars has found it in the cache
				 * and picked up its own reference.
				 */
			}
		}
		sf.Free()
		agefont(f)
		twid += wid
	}
	return twid
}

func (f *Font) StringWidth(s string) int {
	return stringnwidth(f, s, nil, nil)
}

func (f *Font) BytesWidth(b []byte) int {
	return stringnwidth(f, "", b, nil)
}

func (f *Font) RunesWidth(r []rune) int {
	return stringnwidth(f, "", nil, r)
}

func (f *Font) StringSize(s string) image.Point {
	return image.Pt(f.StringWidth(s), f.Height)
}

func (f *Font) BytesSize(b []byte) image.Point {
	return image.Pt(f.BytesWidth(b), f.Height)
}

func (f *Font) RunesSize(r []rune) image.Point {
	return image.Pt(f.RunesWidth(r), f.Height)
}