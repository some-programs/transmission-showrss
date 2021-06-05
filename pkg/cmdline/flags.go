package cmdline

import (
	"flag"
	"strconv"
	"strings"

	"github.com/some-programs/transmission-showrss/pkg/showrss"
)

// TransmissionConfig .
type TransmissionConfig struct {
	Address  string
	User     string
	Password string
}

func TransmissionConfigFlags(fs *flag.FlagSet) *TransmissionConfig {
	v := &TransmissionConfig{}
	fs.StringVar(&v.Address, "url", "http://localhost:9091/transmission/rpc", "URL to tranmission rpc server")
	fs.StringVar(&v.User, "user", "", "transmission rpc server username")
	fs.StringVar(&v.Password, "pass", "", "transmission rpc server password")
	return v
}

func FeedSelectionFlags(fs *flag.FlagSet) *showrss.FeedSelection {
	// FeedSelectionFlags .
	v := &showrss.FeedSelection{
		Shows: make([]int, 0),
		Users: make([]int, 0),
	}
	fs.Var((*intSliceFlag)(&v.Users), "users", "showrss user id's, comma separated")
	fs.Var((*intSliceFlag)(&v.Shows), "shows", "showrss show id's, comma separated")
	return v
}

func ShowDirsFlags(fs *flag.FlagSet) *showrss.ShowDirs {
	v := &showrss.ShowDirs{}
	fs.StringVar(&v.Path, "path", "../Shows", "transmission downlod directory, absolut or relative to transmissions download-dir setting")
	fs.BoolVar(&v.Dirs, "dirs", true, "add show name to transmission downlod directory: {path}/[show name]/")
	return v
}

// intSliceFlag is a flag type which
type intSliceFlag []int

func (s *intSliceFlag) String() string {
	var nums []string
	for _, n := range *s {
		nums = append(nums, strconv.Itoa(n))
	}
	return strings.Join(nums, ",")
}

func (f *intSliceFlag) Set(value string) error {
	var res intSliceFlag
	for _, s := range strings.Split(value, ",") {
		s = strings.TrimSpace(s)
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		res = append(res, n)
	}
	*f = res
	return nil
}
