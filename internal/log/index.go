package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

// the width constants define the number of bytes that make up each index entry
var (
	offWidth uint64 = 4
	posWidth uint64 = 8
	entWidth        = offWidth + posWidth
)

type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

// create index and save the current size of the file so we
// can track the amount of data in the index file as we add index entries
// Grow the file to max index size before memory-mapping the file and then
// returning the created index to the caller
func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{
		file: f,
	}

	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	idx.size = uint64(fi.Size())
	if err = os.Truncate(
		f.Name(), int64(c.Segment.MaxIndexBytes),
	); err != nil {
		return nil, err
	}

	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}
	return idx, nil
}

// Read takes in an offset and returns the associated record's position in the store.
// The given offset is relative to the segment's base offset (0 is always the offset of the index's first entry
func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	if i.size == 0 {
		return 0, 0, io.EOF
	}
	if in == -1 {
		out = uint32((i.size / entWidth) - 1)
	} else {
		// Calculate the correct byte position
		if uint64(in) >= i.size/entWidth {
			return 0, 0, io.EOF
		}
		out = uint32(in)
	}
	pos = uint64(out) * entWidth
	out = enc.Uint32(i.mmap[pos : pos+offWidth])
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth])
	return out, pos, nil
}

// Write() appends the given offset and position to the index.
// Validate that we have space to write entry.
// If there is space, encode the offset and position and
// write them to the memory-mapped file.
// Then we increment the position where the next write will go.
func (i *index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+entWidth {
		return io.EOF
	}
	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)
	i.size += uint64(entWidth)
	return nil
}

func (i *index) Name() string {
	return i.file.Name()
}

// Close makes sure the memory-mapped file has synced its data to the persisted
// file and that the persisted file has flushed its contents to stable storage
// Then it truncates the persisted file to the amount of data that’s
// actually in it and closes the file
func (i *index) Close() error {
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	if err := i.file.Sync(); err != nil {
		return err
	}
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return i.file.Close()
}
