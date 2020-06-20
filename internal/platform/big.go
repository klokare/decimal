// +build armbe arm64be mips mips64 mips64p32 ppc ppc64 s390 s390x sparc sparc64

package platform

// build tags from https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63

// LittleEndian is false for big endian architectures
const LittleEndian = false
