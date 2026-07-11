package qqm

import (
	"strconv"
	"strings"
)

const (
	keyFrom     = "from"
	keyJoinMode = "join="
	keyJoinTo   = "map="
	keyAlias    = "alias="
	keyUsePK    = "pk"
)

type TableFlags struct {
	IsFrom    bool
	JoinMode  JoinMode
	Alias     string
	RefMap    map[string]string
	UsePk     bool
	SortOrder int
}

type JoinMode int

const (
	JoinModeNone JoinMode = iota
	JoinModeLeft
	JoinModeRight
	JoinModeInner
)

var JoinModeNames = map[string]JoinMode{
	"left":  JoinModeLeft,
	"right": JoinModeRight,
	"inner": JoinModeInner,
}

func parseTableTag(tag string) (result TableFlags, ok bool) {
	keys := strings.Split(tag, keySeparator)
	for _, key := range keys {
		switch {
		case key == keyFrom:
			result.IsFrom = true
		case key == keyUsePK:
			result.UsePk = true
		case isKey(keyOmit, key):
			return result, false // return false !!!
		case isKey(keyJoinMode, key):
			result.JoinMode = JoinModeNames[strings.ToLower(key[len(keyJoinMode):])]
		case isKey(keyAlias, key):
			result.Alias = key[len(keyAlias):]
		case isKey(keyJoinTo, key):
			s := strings.ToLower(key[len(keyJoinTo):])
			ss := strings.Split(s, inKeySeparator)
			result.RefMap = make(map[string]string, len(ss))
			for _, kv := range ss {
				split := strings.SplitN(kv, inKVSeparator, 2)
				if len(split) == 2 {
					result.RefMap[split[0]] = split[1]
				}
			}
		case isKey(keySort, key):
			result.SortOrder, _ = strconv.Atoi(key[len(keySort):])
		}
	}
	return result, true
}
