package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

// 数据库配置
type Config struct {
	Ip        string
	Port      string
	User      string
	Password  string
	Database  string
	HisSuffix string
	Tables    []string
}

type TemplateArgs struct {
	HisSuffix string  // 操作记录表后缀
	FromTable Table   // 来源表
	HisTable  Table   // 历史表
	UpdField  []Field // 历史表修改字段
	AddField  []Field // 历史表新增字段
	DelField  []Field // 历史表删除字段
}

type Field struct {
	Name string // 字段名称
	Key  string // 键类型
	Type string // 定义类型
}

type Table struct {
	Name   string  // 表名
	Fields []Field // 字段列表
}

// 生成历史表结构
var HisTable = `
CREATE TABLE {{.FromTable.Name}}_{{.HisSuffix}} (
  ` + "`{{.HisSuffix}}_id`" + ` bigint(20) NOT NULL AUTO_INCREMENT,
  ` + "`{{.HisSuffix}}_type`" + ` varchar(255) DEFAULT NULL,
  ` + "`{{.HisSuffix}}_date`" + ` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
{{FromFields .AddField}}
  PRIMARY KEY (` + "`{{.HisSuffix}}_id`" + `)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4;
`

// 历史表修改字段
var HisUpdFiled = `{{UpdFiled .}}`

// 历史表新增字段
var HisAddFiled = `{{AddFiled .}}`

// 历史表删除字段
var HisDelFiled = `{{DelFiled .}}`

// 生成插入数据触发器
var InsertTK = `
CREATE TRIGGER {{.FromTable.Name}}_insert_tk AFTER INSERT
ON {{.FromTable.Name}} FOR EACH ROW
BEGIN
    INSERT INTO {{.FromTable.Name}}_{{.HisSuffix}}(` + "`{{.HisSuffix}}_type`" + `, {{Column "" .FromTable.Fields}}) VALUES ('insert', {{Column "NEW." .FromTable.Fields}});
END;
`

// 生成更新表触发器
var UpdateTK = `
CREATE TRIGGER {{.FromTable.Name}}_update_tk AFTER UPDATE
ON {{.FromTable.Name}} FOR EACH ROW
BEGIN
    INSERT INTO {{.FromTable.Name}}_{{.HisSuffix}}(` + "`{{.HisSuffix}}_type`" + `, {{Column "" .FromTable.Fields}}) VALUES ('update', {{Column "NEW." .FromTable.Fields}});
END;
`

// 生成删除表触发器
var DeleteTK = `
CREATE TRIGGER {{.FromTable.Name}}_delete_tk BEFORE DELETE
ON {{.FromTable.Name}} FOR EACH ROW
BEGIN
    INSERT INTO {{.FromTable.Name}}_{{.HisSuffix}}(` + "`{{.HisSuffix}}_type`" + `, {{Column "" .FromTable.Fields}}) VALUES ('delete', {{Column "OLD." .FromTable.Fields}});
END;
`

const ConfigTemplate = `
{
	"Ip": "127.0.0.1",
	"Port": "3306",
	"User": "root",
	"Password": "root",
	"Database": "mydatabase",
	"HisSuffix": "his",
	"Tables": ["hello", "test"]
}
`

func Column(prefix string, args []Field) string {
	var temp []string
	for _, v := range args {
		temp = append(temp, prefix+"`"+v.Name+"`")
	}
	return strings.Join(temp, ", ")
}

// 来源表字段
func FromFields(args []Field) string {
	var temp []string
	for i := 0; i < len(args); i++ {
		temp = append(temp, "  `"+args[i].Name+"` "+args[i].Type)
	}
	return strings.Join(temp, ",\n") + ","
}

// 修改字段
func UpdFiled(args TemplateArgs) string {
	var res = ""
	for _, v := range args.UpdField {
		res += "ALTER TABLE " + args.FromTable.Name + "_" + args.HisSuffix + " MODIFY COLUMN `" + v.Name + "` " + v.Type + ";\n"
	}
	return res
}

// 添加字段
func AddFiled(args TemplateArgs) string {
	var res = ""
	for _, v := range args.AddField {
		res += "ALTER TABLE " + args.FromTable.Name + "_" + args.HisSuffix + " ADD COLUMN `" + v.Name + "` " + v.Type + ";\n"
	}
	return res
}

// 删除字段
func DelFiled(args TemplateArgs) string {
	var res = ""
	for _, v := range args.DelField {
		res += "ALTER TABLE " + args.FromTable.Name + "_" + args.HisSuffix + " DROP COLUMN `" + v.Name + "`;\n"
	}
	return res
}

// 修改的字段
func UpdateSub(src *Table, dst *Table) []Field {
	if src == nil {
		return nil
	}
	if dst == nil {
		return src.Fields
	}
	// 判断是否存在修改
	var has = func(k Field) bool {
		for _, s := range dst.Fields {
			if s.Name == k.Name && s.Type != k.Type {
				return true
			}
		}
		return false
	}
	var res []Field
	for _, s := range src.Fields {
		if has(s) {
			res = append(res, s)
		}
	}
	return res
}

// 找出src表中存在而dst表不存在的字段
func Sub(src *Table, dst *Table) []Field {
	if src == nil {
		return nil
	}
	if dst == nil {
		return src.Fields
	}
	// 判断是否存在
	var has = func(k Field) bool {
		for _, s := range dst.Fields {
			if s.Name == k.Name {
				return true
			}
		}
		return false
	}
	var res []Field
	for _, s := range src.Fields {
		if !has(s) {
			res = append(res, s)
		}
	}
	return res
}

func main() {
	var conf string    // 配置文件
	var outconfig bool // 打印配置文件模板
	var out io.Writer  // 输出流
	var outfile string // 输出文件
	flag.StringVar(&conf, "c", "config.json", "Config file path.")
	flag.BoolVar(&outconfig, "gc", false, "Print config template.")
	flag.StringVar(&outfile, "o", "", "Output file.")
	flag.Parse()
	if outfile == "" {
		out = os.Stdout
	} else {
		file, err := os.Create(outfile)
		if err != nil {
			return
		}
		defer file.Close()
		out = file
	}
	if outconfig {
		out.Write([]byte(ConfigTemplate))
		return
	}
	f, err := ioutil.ReadFile(conf)
	if err != nil {
		return
	}
	var config Config
	err = json.Unmarshal(f, &config)
	if err != nil {
		fmt.Println(err)
		return
	}

	var host = config.User + ":" + config.Password + "@tcp(" + config.Ip + ":" + config.Port + ")/" + config.Database + "?charset=utf8"
	con, err := sql.Open("mysql", host)
	defer con.Close()
	if err != nil {
		return
	}
	var getTable = func(tb string, db string) *Table {
		var temp *Table
		rows, err := con.Query("select c.COLUMN_NAME, c.COLUMN_KEY, c.COLUMN_TYPE from information_schema.COLUMNS c where c.TABLE_NAME = ? and c.TABLE_SCHEMA = ?", tb, db)
		if err != nil {
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			if temp == nil {
				temp = &Table{}
				temp.Name = tb
			}
			var field Field
			rows.Scan(&field.Name, &field.Key, &field.Type)
			temp.Fields = append(temp.Fields, field)
		}
		return temp
	}
	tpl := template.New("main")
	tpl.Funcs(template.FuncMap{"Column": Column})
	tpl.Funcs(template.FuncMap{"UpdFiled": UpdFiled})
	tpl.Funcs(template.FuncMap{"AddFiled": AddFiled})
	tpl.Funcs(template.FuncMap{"DelFiled": DelFiled})
	tpl.Funcs(template.FuncMap{"FromFields": FromFields})
	for _, table := range config.Tables {
		out.Write([]byte("\n-- " + table + "\n"))
		var fromTable = getTable(table, config.Database)                     // 原始表
		var hisTable = getTable(table+"_"+config.HisSuffix, config.Database) // 历史记录表
		var args TemplateArgs
		args.HisSuffix = config.HisSuffix
		args.FromTable = *fromTable
		if hisTable == nil {
			// 历史表不存在
			args.AddField = Sub(fromTable, nil)
			tab, err := tpl.Parse(HisTable)
			if err != nil {
				return
			}
			tab.Execute(out, args)
		} else {
			args.HisTable = *hisTable
			// 历史表修改字段处理
			var upd = UpdateSub(fromTable, hisTable)
			args.UpdField = upd
			upf, err := tpl.Parse(HisUpdFiled)
			if err != nil {
				return
			}
			upf.Execute(out, args)
			// 排除历史表固定字段，将历史表存在而基础表不存在的字段删除
			var del = Sub(hisTable, fromTable)
			if del != nil {
				// 有删除项
				// 排除历史表固定字段
				var dels []Field
				for _, v := range del {
					if v.Name == args.HisSuffix+"_id" ||
						v.Name == args.HisSuffix+"_type" ||
						v.Name == args.HisSuffix+"_date" {
						continue
					}
					dels = append(dels, v)
				}
				args.DelField = dels
			}
			// 将历史表存在而基础表不存在的字段删除
			def, err := tpl.Parse(HisDelFiled)
			if err != nil {
				return
			}
			def.Execute(out, args)
			// 将基础表中存在而历史表不存在的字段新增到历史表
			args.AddField = Sub(fromTable, hisTable)
			adf, err := tpl.Parse(HisAddFiled)
			if err != nil {
				return
			}
			adf.Execute(out, args)
		}
		// 找出存在的触发器，删掉
		rows, err := con.Query("SELECT t.TRIGGER_NAME FROM information_schema.`TRIGGERS` t where t.EVENT_OBJECT_SCHEMA = ? AND t.EVENT_OBJECT_TABLE = ?", config.Database, table)
		if err != nil {
			return
		}
		for rows.Next() {
			var name string
			rows.Scan(&name)
			if name == table+"_insert_tk" ||
				name == table+"_update_tk" ||
				name == table+"_delete_tk" {
				out.Write([]byte("DROP TRIGGER " + name + ";\n"))
			}
		}
		// 新增触发器
		// insert
		ins, err := tpl.Parse(InsertTK)
		if err != nil {
			return
		}
		ins.Execute(out, args)
		// update
		upd, err := tpl.Parse(UpdateTK)
		if err != nil {
			return
		}
		upd.Execute(out, args)
		// delete
		det, err := tpl.Parse(DeleteTK)
		if err != nil {
			return
		}
		det.Execute(out, args)
	}
	if outfile == "" {
		fmt.Scanf("\n")
	}
}
