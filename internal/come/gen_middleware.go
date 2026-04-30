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
	sb.WriteString("\t\"sync\"\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(`func CORS(origin string)func(http.Handler)http.Handler{
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			w.Header().Set("Access-Control-Allow-Origin",origin)
			w.Header().Set("Access-Control-Allow-Methods","GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers","Content-Type,Authorization,X-API-Version")
			if r.Method=="OPTIONS"{
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w,r)
		})
	}
}

`)

	sb.WriteString(`type responseRecorder struct{
	http.ResponseWriter
	status int
}

func (r *responseRecorder)WriteHeader(code int){
	r.status=code
	r.ResponseWriter.WriteHeader(code)
}

func Logging()func(http.Handler)http.Handler{
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			start:=time.Now()
			rec:=&responseRecorder{ResponseWriter:w,status:200}
			next.ServeHTTP(rec,r)
			log.Printf("%s %s %d %v",r.Method,r.URL.Path,rec.status,time.Since(start))
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

	sb.WriteString(`type rateLimiter struct{
	mu       sync.Mutex
	visitors map[string]*visitorInfo
	limit    int
	window   time.Duration
}

type visitorInfo struct{
	count   int
	resetAt time.Time
}

func (rl *rateLimiter)allow(key string)bool{
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now:=time.Now()
	v,ok:=rl.visitors[key]
	if !ok||now.After(v.resetAt){
		rl.visitors[key]=&visitorInfo{count:1,resetAt:now.Add(rl.window)}
		return true
	}
	v.count++
	return v.count<=rl.limit
}

func RateLimit(limit int,window time.Duration)func(http.Handler)http.Handler{
	rl:=&rateLimiter{visitors:make(map[string]*visitorInfo),limit:limit,window:window}
	cleanup:=func(){
		for{
			time.Sleep(time.Minute)
			rl.mu.Lock()
			now:=time.Now()
			for k,v:=range rl.visitors{
				if now.After(v.resetAt){
					delete(rl.visitors,k)
				}
			}
			rl.mu.Unlock()
		}
	}
	go cleanup()
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			key:=r.RemoteAddr
			if !rl.allow(key){
				w.Header().Set("Content-Type","application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w,r)
		})
	}
}

`)

	sb.WriteString(`func APIVersion(requiredVersion string)func(http.Handler)http.Handler{
	return func(next http.Handler)http.Handler{
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			v:=r.Header.Get("X-API-Version")
			if v==""{
				Error(w,http.StatusBadRequest,"API version header required")
				return
			}
			if v!=requiredVersion{
				Error(w,http.StatusBadRequest,"unsupported API version")
				return
			}
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

func InvalidQueryParams(w http.ResponseWriter){
	Error(w,http.StatusBadRequest,"Invalid query parameters")
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
