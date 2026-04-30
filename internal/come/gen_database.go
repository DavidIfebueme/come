package come

import (
	"strings"
)

func GenDatabase(proj *Project) string {
	var sb strings.Builder
	sb.WriteString("package database\n\nimport(\n")
	sb.WriteString("\t\"database/sql\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"strings\"\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString("\t_ \"github.com/lib/pq\"\n")
	sb.WriteString("\t_ \"modernc.org/sqlite\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("type DB struct{\n")
	sb.WriteString("\t*sql.DB\n")
	sb.WriteString("\tdriver string\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func Connect(url string)(*DB,func(),error){\n")
	sb.WriteString("\tdriver:=detectDriver(url)\n")
	sb.WriteString("\tdb,err:=sql.Open(driver,url)\n")
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\treturn nil,nil,fmt.Errorf(\"open db: %w\",err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdb.SetMaxOpenConns(25)\n")
	sb.WriteString("\tdb.SetMaxIdleConns(5)\n")
	sb.WriteString("\tdb.SetConnMaxLifetime(5*time.Minute)\n")
	sb.WriteString("\tcleanup:=func(){db.Close()}\n")
	sb.WriteString("\treturn &DB{DB:db,driver:driver},cleanup,nil\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func detectDriver(url string)string{\n")
	sb.WriteString("\tif strings.HasPrefix(url,\"postgres://\")||strings.HasPrefix(url,\"postgresql://\"){\n")
	sb.WriteString("\t\treturn \"postgres\"\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn \"sqlite\"\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func (db *DB)Driver()string{return db.driver}\n\n")

	sb.WriteString("func (db *DB)Rebind(query string)string{\n")
	sb.WriteString("\tif db.driver!=\"postgres\"{\n")
	sb.WriteString("\t\treturn rebindDollar(query)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn query\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func rebindDollar(query string)string{\n")
	sb.WriteString("\tvar sb strings.Builder\n")
	sb.WriteString("\tn:=1\n")
	sb.WriteString("\tfor i:=0;i<len(query);i++{\n")
	sb.WriteString("\t\tif query[i]=='$'&&i+1<len(query)&&query[i+1]>='0'&&query[i+1]<='9'{\n")
	sb.WriteString("\t\t\tsb.WriteByte('?')\n")
	sb.WriteString("\t\t\ti++\n")
	sb.WriteString("\t\t\tfor i<len(query)&&query[i]>='0'&&query[i]<='9'{\n")
	sb.WriteString("\t\t\t\ti++\n")
	sb.WriteString("\t\t\t}\n")
	sb.WriteString("\t\t\ti--\n")
	sb.WriteString("\t\t\tn++\n")
	sb.WriteString("\t\t}else{\n")
	sb.WriteString("\t\t\tsb.WriteByte(query[i])\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn sb.String()\n")
	sb.WriteString("}\n")

	return sb.String()
}
