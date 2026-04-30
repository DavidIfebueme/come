package come

import (
	"fmt"
	"strings"
)

func GenMiddleware(proj *Project) string {
	var sb strings.Builder
	sb.WriteString("package server\n\nimport(\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(`func CORS(origin string)func(http.Handler)http.Handler{
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			w.Header().Set("Access-Control-Allow-Origin",origin)
			w.Header().Set("Access-Control-Allow-Methods","GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers","Content-Type,Authorization")
			if r.Method=="OPTIONS"{
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w,r)
		})
	}
}

`)

	sb.WriteString(`func Logging()func(http.Handler)http.Handler{
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			start:=time.Now()
			next.ServeHTTP(w,r)
			log.Printf("%s %s %v",r.Method,r.URL.Path,time.Since(start))
		})
	}
}

`)

	sb.WriteString(`func Recovery()func(http.Handler)http.Handler{
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			defer func(){
				if err:=recover();err!=nil{
					log.Printf("panic: %v",err)
					Error(w,http.StatusInternalServerError,"internal server error")
				}
			}()
			next.ServeHTTP(w,r)
		})
	}
}
`)

	return sb.String()
}

func GenResponse(proj *Project) string {
	var sb strings.Builder
	sb.WriteString("package server\n\nimport(\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(`func Error(w http.ResponseWriter,status int,message string){
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"status":"error","message":message})
}

func ValidationErrors(w http.ResponseWriter,errs[]string){
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(map[string]any{"status":"error","message":"validation failed","errors":errs})
}

func Unauthorized(w http.ResponseWriter){
	Error(w,http.StatusUnauthorized,"unauthorized")
}

func Forbidden(w http.ResponseWriter){
	Error(w,http.StatusForbidden,"forbidden")
}
`)

	_ = fmt.Sprintf("%s", strings.Join(nil, ""))
	return sb.String()
}
