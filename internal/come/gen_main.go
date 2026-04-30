package come

import (
	"fmt"
	"strings"
)

func GenMain(proj *Project) string {
	var sb strings.Builder
	sb.WriteString("package main\n\nimport(\n")
	sb.WriteString(fmt.Sprintf("\t\"%s/pkg/database\"\n", proj.AppName))
	sb.WriteString(fmt.Sprintf("\t\"%s/pkg/server\"\n", proj.AppName))
	if proj.Bouncer != nil {
		sb.WriteString(fmt.Sprintf("\tjwtauth %q\n", proj.AppName+"/pkg/auth"))
	}
	for _, feat := range proj.Features {
		alias := feat.Name
		if proj.Bouncer != nil && feat.Name == "auth" {
			alias = "authapi"
		}
		sb.WriteString(fmt.Sprintf("\t%s %q\n", alias, proj.AppName+"/internal/"+feat.Name))
	}
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"os\"\n")
	sb.WriteString("\t\"os/signal\"\n")
	sb.WriteString("\t\"syscall\"\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("func main(){\n")

	sb.WriteString("\tport:=os.Getenv(\"PORT\")\n")
	sb.WriteString("\tif port==\"\"{\n")
	sb.WriteString(fmt.Sprintf("\t\tport=\"%d\"\n", proj.Aura.Port))
	sb.WriteString("\t}\n\n")

	sb.WriteString("\tdbURL:=os.Getenv(\"DATABASE_URL\")\n")
	if len(proj.DBs) > 0 {
		sb.WriteString("\tif dbURL==\"\"{\n")
		sb.WriteString(fmt.Sprintf("\t\tdbURL=%q\n", proj.DBs[0].Connection))
		sb.WriteString("\t}\n")
	}
	sb.WriteString("\tdb,cleanup,err:=database.Connect(dbURL)\n")
	sb.WriteString("\tif err!=nil{\n")
	sb.WriteString("\t\tlog.Fatal(err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdefer cleanup()\n\n")

	if proj.Bouncer != nil {
		sb.WriteString("\tauthCfg:=jwtauth.Config{\n")
		sb.WriteString(fmt.Sprintf("\t\tSecret:os.Getenv(\"JWT_SECRET\"),\n"))
		sb.WriteString(fmt.Sprintf("\t\tExpire:%s,\n", durationToGo(proj.Bouncer.Expire)))
		sb.WriteString("\t}\n")
		sb.WriteString("\tjwtAuth:=jwtauth.New(authCfg)\n\n")
	}

	sb.WriteString("\tmux:=http.NewServeMux()\n\n")

	for _, feat := range proj.Features {
		alias := feat.Name
		if proj.Bouncer != nil && feat.Name == "auth" {
			alias = "authapi"
		}
		sb.WriteString(fmt.Sprintf("\t%sRepo:=%s.NewRepository(db)\n", feat.Name, alias))
		sb.WriteString(fmt.Sprintf("\t%sHandler:=%s.NewHandler(%sRepo", feat.Name, alias, feat.Name))
		if proj.Bouncer != nil {
			sb.WriteString(",jwtAuth")
		}
		sb.WriteString(")\n")
		sb.WriteString(fmt.Sprintf("\t%s.RegisterRoutes(mux,%sHandler)\n\n", alias, feat.Name))
	}

	sb.WriteString("\thandler:=server.Chain(\n")
	sb.WriteString("\t\tmux,\n")
	sb.WriteString("\t\tserver.Logging(),\n")
	sb.WriteString("\t\tserver.Recovery(),\n")
	if proj.CORS.Origin != "" {
		sb.WriteString(fmt.Sprintf("\t\tserver.CORS(%q),\n", proj.CORS.Origin))
	}
	sb.WriteString("\t)\n\n")

	sb.WriteString(fmt.Sprintf("\tsrv:=&http.Server{\n"))
	sb.WriteString(fmt.Sprintf("\t\tAddr:\":\"+port,\n"))
	sb.WriteString(fmt.Sprintf("\t\tHandler:handler,\n"))
	if proj.Aura.ReadTimeout != "" {
		sb.WriteString(fmt.Sprintf("\t\tReadTimeout:%s,\n", durationToGo(proj.Aura.ReadTimeout)))
	}
	if proj.Aura.WriteTimeout != "" {
		sb.WriteString(fmt.Sprintf("\t\tWriteTimeout:%s,\n", durationToGo(proj.Aura.WriteTimeout)))
	}
	if proj.Aura.IdleTimeout != "" {
		sb.WriteString(fmt.Sprintf("\t\tIdleTimeout:%s,\n", durationToGo(proj.Aura.IdleTimeout)))
	}
	sb.WriteString("\t}\n\n")

	sb.WriteString("\tgo func(){\n")
	sb.WriteString("\t\tfmt.Printf(\"server running on :%s\\\\n\",port)\n")
	sb.WriteString("\t\tif err:=srv.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{\n")
	sb.WriteString("\t\t\tlog.Fatal(err)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}()\n\n")

	sb.WriteString("\tquit:=make(chan os.Signal,1)\n")
	sb.WriteString("\tsignal.Notify(quit,syscall.SIGINT,syscall.SIGTERM)\n")
	sb.WriteString("\t<-quit\n\n")

	sb.WriteString("\tfmt.Println(\"shutting down...\")\n")
	sb.WriteString("\tctx,cancel:=context.WithTimeout(context.Background(),10*time.Second)\n")
	sb.WriteString("\tdefer cancel()\n")
	sb.WriteString("\tsrv.Shutdown(ctx)\n")
	sb.WriteString("}\n")

	return sb.String()
}

func durationToGo(d string) string {
	if d == "" {
		return "0"
	}
	multiplier := "time.Second"
	val := d
	if strings.HasSuffix(d, "h") {
		multiplier = "time.Hour"
		val = strings.TrimSuffix(d, "h")
	} else if strings.HasSuffix(d, "m") {
		multiplier = "time.Minute"
		val = strings.TrimSuffix(d, "m")
	} else if strings.HasSuffix(d, "ms") {
		multiplier = "time.Millisecond"
		val = strings.TrimSuffix(d, "ms")
	} else if strings.HasSuffix(d, "s") {
		multiplier = "time.Second"
		val = strings.TrimSuffix(d, "s")
	}
	if val == "1" {
		return multiplier
	}
	return val + "*" + multiplier
}
