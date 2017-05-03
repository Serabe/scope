package process

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	linuxproc "github.com/c9s/goprocinfo/linux"

	"github.com/weaveworks/common/fs"
	"github.com/weaveworks/scope/probe/host"
)

type walker struct {
	procRoot string
}

// NewWalker creates a new process Walker.
func NewWalker(procRoot string) Walker {
	return &walker{procRoot: procRoot}
}

// skipNSpaces skips nSpaces in buf and updates the cursor 'pos'
func skipNSpaces(buf *[]byte, pos *int, nSpaces int) {
	for spaceCount := 0; *pos < len(*buf) && spaceCount < nSpaces; *pos++ {
		if (*buf)[*pos] == ' ' {
			spaceCount++
		}
	}
}

// parseIntWithSpaces is similar to strconv.ParseInt but stops parsing when
// reading a space instead of returning an error
func parseIntWithSpaces(buf *[]byte, pos *int) (ret int) {
	for ; *pos < len(*buf) && (*buf)[*pos] != ' '; *pos++ {
		ret = ret*10 + int((*buf)[*pos]-'0')
	}
	return
}

// parseUint64WithSpaces is similar to strconv.ParseUint64 but stops parsing
// when reading a space instead of returning an error
func parseUint64WithSpaces(buf *[]byte, pos *int) (ret uint64) {
	for ; *pos < len(*buf) && (*buf)[*pos] != ' '; *pos++ {
		ret = ret*10 + uint64((*buf)[*pos]-'0')
	}
	return
}

// readStats reads and parses '/proc/<pid>/stat' files
func readStats(path string) (ppid, threads int, jiffies, rss, rssLimit uint64, err error) {
	const (
		// /proc/<pid>/stat field positions, counting from zero
		// see "man 5 proc"
		procStatFieldPpid        int = 3
		procStatFieldUserJiffies int = 13
		procStatFieldSysJiffies  int = 14
		procStatFieldThreads     int = 19
		procStatFieldRssPages    int = 23
		procStatFieldRssLimit    int = 24
	)
	var (
		buf                               []byte
		userJiffies, sysJiffies, rssPages uint64
	)
	buf, err = fs.ReadFile(path)
	if err != nil {
		return
	}

	// Parse the file without using expensive extra string allocations

	pos := 0
	skipNSpaces(&buf, &pos, procStatFieldPpid)
	ppid = parseIntWithSpaces(&buf, &pos)

	skipNSpaces(&buf, &pos, procStatFieldUserJiffies-procStatFieldPpid)
	userJiffies = parseUint64WithSpaces(&buf, &pos)

	pos++ // 1 space between userJiffies and sysJiffies
	sysJiffies = parseUint64WithSpaces(&buf, &pos)

	skipNSpaces(&buf, &pos, procStatFieldThreads-procStatFieldSysJiffies)
	threads = parseIntWithSpaces(&buf, &pos)

	skipNSpaces(&buf, &pos, procStatFieldRssPages-procStatFieldThreads)
	rssPages = parseUint64WithSpaces(&buf, &pos)

	pos++ // 1 space between rssPages and rssLimit
	rssLimit = parseUint64WithSpaces(&buf, &pos)

	jiffies = userJiffies + sysJiffies
	rss = rssPages * uint64(os.Getpagesize())
	return
}

func readLimits(path string) (openFilesLimit uint64, err error) {
	buf, err := cachedReadFile(path)
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(buf), "\n") {
		if strings.HasPrefix(line, "Max open files") {
			splits := strings.Fields(line)
			if len(splits) < 6 {
				return 0, fmt.Errorf("Invalid /proc/PID/limits")
			}
			openFilesLimit, err := strconv.Atoi(splits[3])
			return uint64(openFilesLimit), err
		}
	}
	return 0, nil
}

// Walk walks the supplied directory (expecting it to look like /proc)
// and marshalls the files into instances of Process, which it then
// passes one-by-one to the supplied function. Walk is only made public
// so that is can be tested.
func (w *walker) Walk(f func(Process, Process)) error {
	dirEntries, err := fs.ReadDirNames(w.procRoot)
	if err != nil {
		return err
	}

	for _, filename := range dirEntries {
		pid, err := strconv.Atoi(filename)
		if err != nil {
			continue
		}

		ppid, threads, jiffies, rss, rssLimit, err := readStats(path.Join(w.procRoot, filename, "stat"))
		if err != nil {
			continue
		}

		openFilesCount, err := fs.ReadDirCount(path.Join(w.procRoot, filename, "fd"))
		if err != nil {
			continue
		}

		openFilesLimit, err := readLimits(path.Join(w.procRoot, filename, "limits"))
		if err != nil {
			continue
		}

		cmdline, name := "", ""
		if cmdlineBuf, err := cachedReadFile(path.Join(w.procRoot, filename, "cmdline")); err == nil {
			// like proc, treat name as the first element of command line
			i := bytes.IndexByte(cmdlineBuf, '\000')
			if i == -1 {
				i = len(cmdlineBuf)
			}
			name = string(cmdlineBuf[:i])
			cmdlineBuf = bytes.Replace(cmdlineBuf, []byte{'\000'}, []byte{' '}, -1)
			cmdline = string(cmdlineBuf)
		}
		if name == "" {
			if commBuf, err := cachedReadFile(path.Join(w.procRoot, filename, "comm")); err == nil {
				name = "[" + strings.TrimSpace(string(commBuf)) + "]"
			} else {
				name = "(unknown)"
			}
		}
		f(Process{
			PID:            pid,
			PPID:           ppid,
			Name:           name,
			Cmdline:        cmdline,
			Threads:        threads,
			Jiffies:        jiffies,
			RSSBytes:       rss,
			RSSBytesLimit:  rssLimit,
			OpenFilesCount: openFilesCount,
			OpenFilesLimit: openFilesLimit,
		}, Process{})
	}

	return nil
}

var previousStat = linuxproc.CPUStat{}

// GetDeltaTotalJiffies returns the number of jiffies that have passed since it
// was last called.  In that respect, it is side-effect-ful.
func GetDeltaTotalJiffies() (uint64, float64, error) {
	stat, err := linuxproc.ReadStat(host.ProcStat)
	if err != nil {
		return 0, 0.0, err
	}

	var (
		currentStat = stat.CPUStatAll
		prevTotal   = (previousStat.Idle + previousStat.IOWait + previousStat.User +
			previousStat.Nice + previousStat.System + previousStat.IRQ +
			previousStat.SoftIRQ + previousStat.Steal)
		currentTotal = (currentStat.Idle + currentStat.IOWait + currentStat.User +
			currentStat.Nice + currentStat.System + currentStat.IRQ +
			currentStat.SoftIRQ + currentStat.Steal)
	)
	previousStat = currentStat
	return currentTotal - prevTotal, float64(len(stat.CPUStats)) * 100., nil
}
