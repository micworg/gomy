// ---------------------------------------------------------------------------- gomy - mysql rest api (mwx'2023)
package main

import (
  "os"
  "flag"
  "strconv"
  "net/http"
  "errors"
  "regexp"
  "fmt"
  "log"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "github.com/tidwall/sjson"
  "database/sql"
  _ "github.com/go-sql-driver/mysql"
  fqdn "github.com/Showmax/go-fqdn"
  "github.com/spf13/viper"
  "github.com/gin-gonic/autotls"
) 

func main() { // ------------------------------------------------------------------------------------------ main
  
  opt_s := flag.Bool("s", false, "setup database and config")
  opt_r := flag.Bool("r", false, "start gomy service")
  opt_p := flag.Int("p", 843, "port on which gomy listens")

  flag.Usage = func() {
    PF("%s (%s (%s), mwx'2023)\n",Cwb("gomy - mysql rest api"),Cgb("v"+version),Cgb(build)) 
    PF("Use '%s' to setup or '%s' to run\n",Cy("gomy -s"),Cy("gomy -r")) 
  }

  flag.Parse()

  if (*opt_s) {                                                                           // setup db and config
    setupdb() 
    os.Exit(0)
  }

  if (*opt_r) {                                                                            // start gomy service

    gin.ForceConsoleColor()
    gin.SetMode(gin.ReleaseMode)
  
    viper.SetDefault("db", "gomy:676f6d79@tcp(localhost:3306)/gomy")
    viper.SetDefault("port", "843")
    viper.SetDefault("chain", "/etc/letsencrypt/live/{FQDN}/fullchain.pem")
    viper.SetDefault("key", "/etc/letsencrypt/live/{FQDN}/privkey.pem")
  
    viper.AddConfigPath("$HOME")
    viper.SetConfigName(".gomy")
    viper.SetConfigType("json")

    err := viper.ReadInConfig()
    if err != nil {
      P(Crb("Failed to open config file."))
      os.Exit(0)
    }

    r := gin.New()
    r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
      return fmt.Sprintf("%s [%s] %s %s %s %d %s \"%s\" %s\n",
        param.TimeStamp.Format("2006/01/02 15:04:05"),
        Cc(param.ClientIP),Cg(param.Method),Cy(param.Path),param.Request.Proto,
        param.StatusCode,param.Latency,param.Request.UserAgent(),param.ErrorMessage,
      )
    }))
   
    v1 := r.Group("/v1")
    {
      v1.POST("/version",getversion)
      v1.POST("/sql",sqlquery)
      v1.POST("/login",login)
      v1.GET("/ping",ping)
    }
    r.GET("/",home) 
    r.GET("/ping",ping)
  
    mdb,_:=sql.Open("mysql",VS("db"))                                                           // ping database
    err=mdb.Ping()
    if err != nil { P(Crb(err.Error())); os.Exit(1); } 
  
    regex:=regexp.MustCompile("{FQDN}")
    chain:=regex.ReplaceAllString(VS("chain"),fqdn.Get())
    key:=regex.ReplaceAllString(VS("key"),fqdn.Get())

    port:=VS("port");if isflagpassed("p") { port=strconv.Itoa(*opt_p) }
    
    PF("%s (%s (%s), mwx'2023)\n",Cwb("Starting gomy service"),Cg("v"+version),Cg(build)) 
    
    PF("Config: %s\n",Cm(viper.ConfigFileUsed()))
    PF("Database: %s\n",Cc(VS("db")))
    PF("FQDN: %s, port: %s\n",Cc(fqdn.Get()),Cc(VS("port")))


    if (viper.GetBool("autotls")) {
      PF("Test with: '%s'\n",Cy("curl -s https://"+fqdn.Get()+"/ping | jq"))
      err=autotls.Run(r, fqdn.Get())
    } else {
      if (viper.GetBool("tls")) {
        PF("Chain: %s\n",Cm(chain))
        PF("Key: %s\n",Cm(key))
        PF("Test with: '%s'\n",Cy("curl -s https://"+fqdn.Get()+":"+port+"/ping | jq"))
        err=http.ListenAndServeTLS(":"+port,chain,key,r)    
      } else {
        PF("Test with: '%s'\n",Cy("curl -s http://"+fqdn.Get()+":"+port+"/ping | jq"))
        err=http.ListenAndServe(":"+port,r)    
      }
    }
    
    if err!=nil { log.Panic(err) }
  
  }
  
  if (!*opt_r && !*opt_s) {
    flag.Usage()
  }
  
}

func home(c *gin.Context) { // ---------------------------------------------------------------------------- home
  html:="<!DOCTYPE html>\n<HTML><HEAD><meta charset=\"UTF-8\"><TITLE>gomy</TITLE></HEAD>"+
        "<BODY><CENTER><PRE>gomy<BR>mysql rest api<BR>v"+version+", mwx'2023</PRE></CENTER></BODY></HTML>"
   c.Data(200, "text/html; charset=utf-8", []byte(html) )
}

func sqlquery(c *gin.Context) { // ------------------------------------------------------------------- sql query
  J:=um(c);
  token:=getpar(&J,"token");

  user,db,pw,err := tokenauth(c,token)

  if err==nil {                                                                     // authentication successful

    con:=user+":"+pw+"@tcp("+VS("dbhost")+":"+VS("dbport")+")/"+db                        // check db connection
    udb,_:=sql.Open("mysql",con)
    err=udb.Ping()
  
    if err==nil {                                                                    // db connection successful
      sql:=getpar(&J,"sql"); 
      r:=regexp.MustCompile("`")
      sql=r.ReplaceAllString(sql, "'")

      L(c,"SQL",sql)
      rows, err := udb.Query(sql)

      if err == nil {                                                                    // sql query successful
      
        value,_:=sjson.Set("","success",1)
        tab:=mytable(rows)
        if (len(tab)>0) {
          value, _ = sjson.Set(value, "data",tab)
        }
        c.Data(200, "text/plain; charset=utf-8", []byte(value) )

      } else { jfailed(c,err.Error()) } 
    } else { jfailed(c,err.Error()) }
  } else { jfailed(c,err.Error()) }

}

func getversion(c *gin.Context) { // --------------------------------------------------------------- get version
  J:=um(c);
  token:=getpar(&J,"token"); 

  _,_,_,err := tokenauth(c,token)

  if err==nil {                                                                     // authentication successful
    value,_:= sjson.Set("", "success", 1)
    value,_= sjson.Set(value, "info",  "gomy, mysql rest api handler")
    value,_= sjson.Set(value, "author",  "mwx'2023")
    value,_= sjson.Set(value, "version",  version)
    c.Data(200, "text/plain; charset=utf-8", []byte(value))
  } else {
    jfailed(c,err.Error()); return 
  }

}

func ping(c *gin.Context) { // ---------------------------------------------------------------------------- ping
  mdb,_:=sql.Open("mysql",VS("db"))
  err:=mdb.Ping()
  if err == nil { 
    value,_:= sjson.Set("", "success", 1)
    value,_= sjson.Set(value, "ping",  "gomy service is running")
    c.Data(200, "text/plain; charset=utf-8", []byte(value))
  }
}

func login(c *gin.Context) { // ------------------------------------------------------------------ login request
  J:=um(c);
  
  user:=getpar(&J,"user"); 
  db:=getpar(&J,"db"); 
  pw:=getpar(&J,"pw"); 


  mdb,_:=sql.Open("mysql",VS("db"))                                                             // ping database
  err:=mdb.Ping()
  if err != nil { jfailed(c,err.Error()); return; } 
 
  dbs := make(map[string]int)                                                                  // load db access
  rows,err := mdb.Query("SELECT name,allow FROM db ORDER BY id DESC")
  for rows.Next() { var name string; var allow int; rows.Scan(&name, &allow); dbs[name]=allow }

  users := make(map[string]int)                                                              // load user access
  rows,err = mdb.Query("SELECT name,allow FROM user ORDER BY id DESC")
  for rows.Next() { var name string; var allow int; rows.Scan(&name,&allow); users[name]=allow }

  ips := make(map[string]int)                                                                // load user access
  rows,err = mdb.Query("SELECT ip,allow FROM ip ORDER BY id DESC")
  for rows.Next() { var ip string; var allow int; rows.Scan(&ip,&allow); ips[ip]=allow }


  dbmatch:=int(dbs["*"])                                                                   // check for db rules
  for k, v := range dbs {
    if (k!="*") {
      if k==db && v==1 { dbmatch++; }
      if k==db && v==0 { dbmatch--; }
    }
  }
  if (dbmatch<=0) { jfailed(c,"access denied by db rule"); return; }

  usermatch:=int(users["*"])                                                             // check for user rules
  for k, v := range users {
    if (k!="*") {
      if k==user && v==1 { usermatch++; }
      if k==user && v==0 { usermatch--; }
    }
  }
  if (usermatch<=0) { jfailed(c,"access denied by user rule"); return; }
  
  ipmatch:=int(ips["*"])                                                                   // check for ip rules
  for k, v := range ips {
    if (k!="*") {
      if checkip(k,c.ClientIP()) && v==1 { ipmatch++; }
      if checkip(k,c.ClientIP()) && v==0 { ipmatch--; }
    }
  }
  if (ipmatch<=0) { jfailed(c,"access denied by ip rule"); return; }

  con:=user+":"+pw+"@tcp("+VS("dbhost")+":"+VS("dbport")+")/"+db                          // check db connection

  udb, _ := sql.Open("mysql",con)
  err = udb.Ping()

  if err==nil {                                                                      // db connection successful
    token := getid(24);

    mdb.Exec("DELETE FROM auth WHERE ip='"+c.ClientIP()+"'")

    sql := "INSERT INTO auth VALUES(0,now(),'"+c.ClientIP()+"','"+token+"','"+user+"','"+db+"','"+pw+"')"
    _, err := mdb.Exec(sql)

    if err != nil {
      jfailed(c,err.Error()) 
    } else {
      value, _ := sjson.Set("", "success", 1)
      value, _ = sjson.Set(value, "token",  token)
      c.Data(200, "text/plain; charset=utf-8", []byte(value) )
    }
  } else {
    jfailed(c,err.Error()) 
  }
}
 
// ----------------------------------------------------------------------------------------------------- SUPPORT

func L(c *gin.Context,p string, v ...any) { // --------------------------------------------------------- logging
  log.SetPrefix(Cc("["+c.ClientIP()+","+p+"] "))
  log.SetFlags(log.LstdFlags | log.Lmsgprefix | log.Lshortfile) 
  log.Println(v...)
}

func tokenauth(c *gin.Context, token string) (string, string, string, error) { // -------- authenticate by token
  r, _ := regexp.Compile("^\\s*$")
    
  if !r.MatchString(token) {
    if len(token)==24 {
      mdb, _ := sql.Open("mysql",VS("db"))

      rows,err := mdb.Query(
        "SELECT *,unix_timestamp(now())-unix_timestamp(ts) AS dt FROM auth WHERE token='"+token+"'")
      
      if (err==nil) {

        tab:=mytable(rows)
      
        if (len(tab)==0) { return "","","",errors.New("token not found") }
        
        if (len(tab)==1) {
          
          if (c.ClientIP()==tab[0]["ip"]) {
            if atoi(tab[0]["dt"].(string))<86400 {

              L(c,"TOKEN",Cm(token)+" ("+tab[0]["dt"].(string)+")")

              mdb.Exec("UPDATE auth SET ts=now() WHERE ip='"+c.ClientIP()+"'")

              return tab[0]["user"].(string),tab[0]["db"].(string),tab[0]["pw"].(string),nil

            } else { return "","","",errors.New("token expired") }
          } else { return "","","",errors.New("ip does not match") }
        } else { return "","","",errors.New("token count error") }
      } else { return "","","",errors.New(err.Error()) }
    } else { return "","","",errors.New("invalid token length") }
  } else { return "","","",errors.New("empty token") }
}

func getpar(j *map[string]interface{},k string) (string) { // ------------------------------------ get parameter
  v,ok := (*j)[k];
  if ok {
    return v.(string)
  } else { 
    return ""
  }
}
  
func um(c *gin.Context) map[string]interface{} { // ---------------------------- unmarshal json from gin context
  jd, _ := c.GetRawData()
  var J map[string]interface{}
  json.Unmarshal([]byte(jd), &J)
  return J
}  

func jfailed(c *gin.Context,msg string) { // ---------------------------------------------------- request failed
  value, _ := sjson.Set("", "success", 0)
  value, _ = sjson.Set(value, "err", msg)
  c.Data(200, "text/plain; charset=utf-8", []byte(value) )
  L(c,"FAILED",Crb(msg))
} 

func jsuccess(c *gin.Context,msg string) { // ----------------------------------------------- request successful
  value, _ := sjson.Set("", "success", 1)
  value, _ = sjson.Set(value, "message", msg)
  c.Data(200, "text/plain; charset=utf-8", []byte(value) )
}

// --------------------------------------------------------------------------------------------------------- END
