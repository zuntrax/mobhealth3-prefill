package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/kjk/lzmadec"
	"github.com/xwb1989/sqlparser"
)

func statementFilter(r io.Reader) io.Reader {
	// TODO use bufio instead
	in, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	splitter := []byte(";\r\n")
	statements := bytes.Split(in, splitter)
	out := bytes.NewBuffer([]byte{})
	for _, v := range statements {
		if bytes.HasPrefix(v, []byte("INSERT INTO `creature_template`")) {
			out.Write(v)
			out.Write(splitter)
		}
	}
	return out
}

func parseDump(r io.Reader, m *mobHPExtractor) {
	tokens := sqlparser.NewTokenizer(r)
	for {
		stmt, err := sqlparser.ParseNext(tokens)
		if err == io.EOF {
			break
		}
		switch stmt := stmt.(type) {
		case *sqlparser.Insert:
			if stmt.Table.Name.String() == "creature_template" {
				m.handleInsert(stmt)
			}
		}
	}
}

type mobInfo struct {
	Name      string
	MinLevel  int
	MaxLevel  int
	MinHealth int
	MaxHealth int
}

func (m mobInfo) getLevels() []mobLevel {
	if m.MinLevel == m.MaxLevel {
		return []mobLevel{mobLevel{
			Name:   m.Name,
			Level:  m.MinLevel,
			Health: m.MinHealth,
		}}
	}

	res := []mobLevel{}
	slope := float64(m.MaxHealth-m.MinHealth) / float64(m.MaxLevel-m.MinLevel)

	for i := m.MinLevel; i <= m.MaxLevel; i++ {
		res = append(res, mobLevel{
			Name:   m.Name,
			Level:  i,
			Health: m.MinHealth + int(slope*float64(i-m.MinLevel)),
		})
	}

	return res
}

type mobLevel struct {
	Name   string
	Level  int
	Health int
}

func (m mobLevel) Format() string {
	return fmt.Sprintf("[\"%s:%d\"] = %d,", strings.Replace(m.Name, "\"", "\\\"", -1), m.Level, m.Health)
}

type mobHPExtractor struct {
	List []mobInfo
}

func (m *mobHPExtractor) handleInsert(stmt *sqlparser.Insert) {
	for _, v := range stmt.Rows.(sqlparser.Values) {
		mob := mobInfo{}
		for i, w := range v {
			col := stmt.Columns[i].String()
			if col != "name" && col != "minlevel" && col != "maxlevel" && col != "minhealth" && col != "maxhealth" {
				continue
			}

			val := string(w.(*sqlparser.SQLVal).Val)

			if col == "name" {
				mob.Name = val
				continue
			}

			num, err := strconv.Atoi(val)
			if err != nil {
				panic(err)
			}

			switch col {
			case "minlevel":
				mob.MinLevel = num
			case "maxlevel":
				mob.MaxLevel = num
			case "minhealth":
				mob.MinHealth = num
			case "maxhealth":
				mob.MaxHealth = num
			}
		}
		m.List = append(m.List, mob)
	}
}

func (m mobHPExtractor) export(out io.Writer) {
	out.Write([]byte("MobHealth3DB = {\n"))
	for _, v := range m.List {
		for _, w := range v.getLevels() {
			out.Write([]byte(fmt.Sprintf("\t%s\n", w.Format())))
		}
	}
	out.Write([]byte("}"))
}

func getDump(path string) io.ReadCloser {
	if strings.HasSuffix(path, ".sql") {
		in, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		return in
	} else if strings.HasSuffix(path, ".7z") {
		archive, err := lzmadec.NewArchive(path)
		if err != nil {
			panic(err)
		}
		for _, v := range archive.Entries {
			if strings.HasSuffix(v.Path, ".sql") {
				r, err := archive.GetFileReader(v.Path)
				if err != nil {
					panic(err)
				}
				return r
			}
		}
		panic("archive doesn't contain sql files")
	} else {
		panic("unknown file extension, use with .sql or .7z file")
	}

}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: mobhealth3-prefill <PATH>")
		os.Exit(1)
	}

	in := getDump(os.Args[1])
	defer in.Close()

	mobs := mobHPExtractor{}
	parseDump(statementFilter(in), &mobs)

	out, err := os.Create("MobHealth.lua")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	mobs.export(out)
}
