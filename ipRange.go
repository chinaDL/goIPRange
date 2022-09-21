package goIPRange

import (
	"encoding/binary"
	"errors"
	"github.com/gogf/gf/v2/text/gstr"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
)

func IP2long(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

func Long2IP(ipLong uint32) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipLong)
	ip := net.IP(ipByte)
	return ip.String()
}

//192.168.1.1
//192.168.1.1/8
//192.168.1.1,192.168.1.2
//192.168.1.1-192.168.255.255
//192.168.1.*
//192.168.1.1-255

func verifyIPStrFormat(ipStr string) bool {
	re, _ := regexp.Compile(`[^\d./,*-]`)
	if re.MatchString(ipStr) {
		return false
	}
	return true
}

type IPRange struct {
	Start  uint32
	End    uint32
	offset int
}

func (b *IPRange) Do(fn func(uint32) bool) {
	for i := b.Start; i < b.End; i += 1 {
		if !fn(i) {
			break
		}
	}
}

func (b *IPRange) GetAllIPToInt() []uint32 {
	ret := make([]uint32, 0)
	b.Do(func(u uint32) bool {
		ret = append(ret, u)
		return true
	})
	return ret
}

func (b *IPRange) GetAllIPToStr() []string {
	ret := make([]string, 0)
	b.Do(func(u uint32) bool {
		ret = append(ret, Long2IP(u))
		return true
	})
	return ret
}

func (b *IPRange) Count() int {
	return int(b.End - b.Start)
}

func (b *IPRange) Include(ip uint32) bool {
	return b.Start <= ip && b.End >= ip
}

func (b *IPRange) IncludeRange(sIP, eIP uint32) bool {
	return b.Include(sIP) || b.Include(eIP)
}

func ipStrToIPRange(ipStr string) (*IPRange, error) {
	var start, end uint32 = 0, 0
	if gstr.Contains(ipStr, "/") {
		s := strings.Split(ipStr, "/")
		if !gstr.IsNumeric(s[1]) {
			return nil, errors.New("掩码错误")
		}
		mask, err := strconv.Atoi(s[1])
		if err != nil {
			return nil, err
		}
		ip := IP2long(s[0])
		c := 32 - mask
		start = (ip >> c << c) + 1
		end = start + uint32(math.Pow(2, float64(c))-2)

	} else if gstr.Contains(ipStr, "-") {
		if gstr.CountI(ipStr, "-") > 1 {
			return nil, errors.New("范围错误")
		}
		s := strings.Split(ipStr, "-")
		sIP, eIP := s[0], s[1]
		start = IP2long(sIP)
		if start == 0 {
			return nil, errors.New("开始IP地址错误")
		}
		eIP = "." + eIP
		//if gstr.Contains(eIP, ".") {
		tSIP := gstr.SplitAndTrim(sIP, ".")
		tEIP := gstr.SplitAndTrim(eIP, ".")
		if len(tEIP) > 4 {
			return nil, errors.New("结束IP地址错误")
		}
		sIndex := len(tSIP) - 1
		for i := len(tEIP) - 1; i >= 0; i-- {
			tSIP[sIndex] = tEIP[i]
			sIndex--
		}
		end = IP2long(strings.Join(tSIP, "."))
		//} else {
		//	end = Ip2long(eIP)
		//
		//}
	} else if gstr.Contains(ipStr, "*") {
		if gstr.CountI(ipStr, ".") != 3 {
			return nil, errors.New("IP地址格式错误")
		}
		s := gstr.SplitAndTrim(ipStr, ".")
		sIP := make([]string, 4)
		eIP := make([]string, 4)
		for i, v := range s {
			sIP[i] = v
			eIP[i] = v
			if v == "*" {
				sIP[i] = "0"
				eIP[i] = "255"
			}
		}
		start = IP2long(strings.Join(sIP, "."))
		end = IP2long(strings.Join(eIP, "."))

	} else {
		start = IP2long(ipStr)
		end = start + 1
	}
	if start > end {
		return nil, errors.New("开始地址大于结束地址")
	}
	return &IPRange{
		Start:  start,
		End:    end,
		offset: 0,
	}, nil
}

func forReplaceChar(str string, old string, new string) string {
	tStr := str
	for {
		if !gstr.Contains(tStr, old) {
			break
		}
		tStr = strings.ReplaceAll(tStr, old, new)
	}
	return tStr
}

type IPContainer struct {
	IPRanges []*IPRange
}

func (b *IPContainer) Do(fn func(ipRange *IPRange) bool) {
	if b == nil || b.IPRanges == nil || len(b.IPRanges) == 0 {
		return
	}
	for _, ipRange := range b.IPRanges {
		if !fn(ipRange) {
			break
		}
	}
}

func (b *IPContainer) Count() int {
	count := 0
	b.Do(func(ipRange *IPRange) bool {
		count += ipRange.Count()
		return true
	})
	return count
}

func (b *IPContainer) AllToStr() []string {
	ret := make([]string, 0)
	b.Do(func(ipRange *IPRange) bool {
		ret = append(ret, ipRange.GetAllIPToStr()...)
		return true
	})
	return ret
}

func (b *IPContainer) AllToLong() []uint32 {
	ret := make([]uint32, 0)
	b.Do(func(ipRange *IPRange) bool {
		ret = append(ret, ipRange.GetAllIPToInt()...)
		return true
	})
	return ret
}

func (b *IPContainer) Include(ipStr string) (bool, error) {
	oldContainer, err := ParseIPStr(ipStr)
	if err != nil {
		return false, err
	}

	isContinue := true
	oldContainer.Do(func(oldIpRange *IPRange) bool {

		b.Do(func(ipRange *IPRange) bool {
			isContinue = !ipRange.IncludeRange(oldIpRange.Start, oldIpRange.End)
			return isContinue
		})
		return isContinue
	})
	return !isContinue, nil
}

func ParseIPStr(ipStr string) (*IPContainer, error) {

	ret := make([]*IPRange, 0)
	nIpStr := strings.ReplaceAll(ipStr, " ", "")
	nIpStr = forReplaceChar(nIpStr, "**", "*")
	nIpStr = forReplaceChar(nIpStr, "//", "/")
	nIpStr = forReplaceChar(nIpStr, "--", "--")
	nIpStr = forReplaceChar(nIpStr, "..", ".")

	if !verifyIPStrFormat(nIpStr) {
		return nil, errors.New("ip格式错误")
	}
	ipSplit := gstr.SplitAndTrim(nIpStr+",", ",")
	ips := make([]string, 0)
	for _, v := range ipSplit {
		if v == "" {
			continue
		}
		ips = append(ips, v)
		ipRange, err := ipStrToIPRange(v)
		if err != nil {
			//fmt.Printf("%s err: %s\n", v, err.Error())
			return nil, err
			//continue
		}
		ret = append(ret, ipRange)
	}
	//fmt.Printf("待处理IP: %+v\n", ips)

	return &IPContainer{IPRanges: ret}, nil
}
