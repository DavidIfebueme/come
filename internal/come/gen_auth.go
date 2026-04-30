package come

import (
	"fmt"
	"strings"
)

func GenAuth(proj *Project) string {
	var sb strings.Builder
	sb.WriteString("package auth\n\nimport(\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"strings\"\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString("\t\"github.com/golang-jwt/jwt/v5\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(`type Config struct{
	Secret string
	Expire time.Duration
}

type Claims struct{
	Sub  string ` + "`" + `json:"sub"` + "`" + `
	Role string ` + "`" + `json:"role"` + "`" + `
	jwt.RegisteredClaims
}

type JWTAuth struct{
	secret []byte
	expire time.Duration
}

func New(cfg Config)*JWTAuth{
	expire:=cfg.Expire
	if expire==0{
		expire=24*time.Hour
	}
	return &JWTAuth{secret:[]byte(cfg.Secret),expire:expire}
}

func (a *JWTAuth)Sign(sub,role string)(string,error){
	claims:=Claims{
		Sub:  sub,
		Role: role,
		RegisteredClaims:jwt.RegisteredClaims{
			ExpiresAt:jwt.NewNumericDate(time.Now().Add(a.expire)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token:=jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
	return token.SignedString(a.secret)
}

func (a *JWTAuth)Validate(r *http.Request)(*Claims,error){
	authHeader:=r.Header.Get("Authorization")
	if authHeader==""{
		return nil,fmt.Errorf("missing authorization header")
	}
	parts:=strings.SplitN(authHeader," ",2)
	if len(parts)!=2||parts[0]!="Bearer"{
		return nil,fmt.Errorf("invalid authorization header format")
	}
	tokenStr:=parts[1]
	token,err:=jwt.ParseWithClaims(tokenStr,&Claims{},func(t *jwt.Token)(interface{},error){
		if _,ok:=t.Method.(*jwt.SigningMethodHMAC);!ok{
			return nil,fmt.Errorf("unexpected signing method: %v",t.Header["alg"])
		}
		return a.secret,nil
	})
	if err!=nil{
		return nil,err
	}
	claims,ok:=token.Claims.(*Claims)
	if !ok||!token.Valid{
		return nil,fmt.Errorf("invalid token")
	}
	return claims,nil
}
`)

	_ = fmt.Sprintf("%s", strings.Join(nil, ""))
	return sb.String()
}
