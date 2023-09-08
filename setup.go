package main

import (
  "database/sql"
  "regexp"
  "github.com/spf13/viper"
  fqdn "github.com/Showmax/go-fqdn"
  "os"
  "strings"
  "strconv"
  "bufio"
)

func setupdb() {
  s:=`
    DROP DATABASE IF EXISTS gomy;
    CREATE DATABASE gomy;
    
    USE gomy;
    
    CREATE TABLE auth (
      id int(11) unsigned NOT NULL AUTO_INCREMENT,
      ts timestamp NULL DEFAULT NULL,
      ip varchar(32) DEFAULT NULL,
      token varchar(64) DEFAULT NULL,
      user varchar(64) DEFAULT NULL,
      db varchar(64) DEFAULT NULL,
      pw varchar(64) DEFAULT NULL,
      PRIMARY KEY (id)
    ) ENGINE=InnoDB AUTO_INCREMENT=26 DEFAULT CHARSET=latin1;
    
    CREATE TABLE user (
      id int(11) unsigned NOT NULL AUTO_INCREMENT,
      name varchar(64) DEFAULT NULL,
      allow int(11) DEFAULT '0',
      PRIMARY KEY (id)
    ) ENGINE=InnoDB DEFAULT CHARSET=latin1;

    CREATE TABLE db (
      id int(11) unsigned NOT NULL AUTO_INCREMENT,
      name varchar(64) DEFAULT NULL,
      allow int(11) DEFAULT '0',
      PRIMARY KEY (id)
    ) ENGINE=InnoDB DEFAULT CHARSET=latin1;

    CREATE TABLE ip (
      id int(11) unsigned NOT NULL AUTO_INCREMENT,
      ip varchar(64) DEFAULT NULL,
      allow int(11) DEFAULT 0,
      PRIMARY KEY (id)
    ) ENGINE=InnoDB CHARSET=latin1;

    CREATE USER 'gomy' IDENTIFIED BY '{PASSWORD}';
    GRANT ALL PRIVILEGES ON gomy.* TO 'gomy';
    
    FLUSH PRIVILEGES;
  `
  gomypw:=getid(16)

  r:=regexp.MustCompile("{PASSWORD}")
  s=r.ReplaceAllString(s,gomypw)

  P(Cwb("Starting gomy database and config setup."))

  viper.AddConfigPath("$HOME")
  viper.SetConfigName(".gomy")
  viper.SetConfigType("json")
  viper.ReadInConfig()

  dbhost:=input("MySQL host:",GD("dbhost","localhost"))
  dbport:=input("MySQL port:",GD("dbport","3306"))
  
  pw:=getmypw()
  
  if (len(pw)>0) {
    usepw:=yesno("Use database password from .my.cnf?",true);
    if (!usepw) {
      pw=""
    }
  } 
  
  if (len(pw)==0) {
    pw=inputpw("MySQL root password:")
  } else {
    P(Cy("Got database password from .my.cnf"))
  }
  
  con:="root:"+pw+"@tcp("+dbhost+":"+dbport+")/mysql?multiStatements=true"  

  db,_:=sql.Open("mysql",con)
  err:=db.Ping()
  if err!=nil {
    P(Crb(err.Error()))
  } else {
    P(Cgb("Databaase connection ok."))

    tls:=yesno("Enable TLS?",VB("tls"));

    chain:="";key:="";port:="";autotls:=false
    if (tls) {
      autotls=yesno("Use autoTLS?",VB("autotls"));
      domain:=input("Full qualified domain name?", fqdn.Get())
      if (!autotls) {
        port=input("The port on which gomy listens?",GD("port","843"))      
        chain=input("Certificate chain file?",GD("chain","/etc/letsencrypt/live/"+domain+"/fullchain.pem"))
        key=input("Certificate key file?",GD("key","/etc/letsencrypt/live/"+domain+"/privkey.pem"))
      } else {
        port="443"
      }
    } else {
      port=input("The port on which gomy listens?","888")
    }
  
    rows,err := db.Query("SELECT count(*) FROM user WHERE User='gomy'")
    if err != nil { P(Crb(err.Error())); os.Exit(0) }
    tab:=mytable(rows)
    if (len(tab)>0) {
      db.Exec("DROP USER 'gomy'")
      if err != nil { P(err.Error()); os.Exit(0) }
    } 
  
    _, err = db.Exec(s)
    if err != nil { P(Crb(err.Error())); os.Exit(0) }

    permode:=yesno("Use default permissions?",true);
    if (permode) {
      setaccess(db,"user","*",1)
      setaccess(db,"user","root",0)
      setaccess(db,"db",  "*",1)
      setaccess(db,"db",  "mysql,sys,information_schema,performance_schema",0)
      setaccess(db,"ip",  "*",1)
      setaccess(db,"ip",  "",0)
    } else {
      setaccess(db,"user",input("Users allowed     :","*"),1)
      setaccess(db,"user",input("Users denied      :","root"),0)
      setaccess(db,"db",  input("Databases allowed :","*"),1)
      setaccess(db,"db",  input("Databases denied  :","mysql,sys,information_schema,performance_schema"),0)
      setaccess(db,"ip",  input("IP's allowed      :","*"),1)
      setaccess(db,"ip",  input("IP's denied       :",""),0)
    }

    gomycon:="gomy:"+gomypw+"@tcp("+dbhost+":"+dbport+")/gomy"  
  
    viper.AddConfigPath("$HOME")
    viper.SetConfigName(".gomy") 
    viper.SetConfigType("json")
    
    viper.Set("tls", tls)
    viper.Set("autotls", autotls)
    viper.Set("db", gomycon) 
    viper.Set("port", port)
    viper.Set("chain", chain)
    viper.Set("key", key)
    viper.Set("dbport", dbport)
    viper.Set("dbhost", dbhost)
     
    homedir,_:=os.UserHomeDir()
    config:=homedir+"/.gomy"
    viper.WriteConfigAs(config) 

    PF("Config written to %s\n",Cm(config))
    
    PF("Start with: '%s'\n",Cy("gomy -r"))
    P(Cgb("Gomy database and config setup done."))

  }
}

func GD(key string,def string) string { // --------------------------------------------------- get default value
  if (len(VS(key))==0) {
    return def
  }
  return VS(key) 
}

func setaccess(db *sql.DB,tab string,access string,mode int) { // ----------------------------------- set access
  s := regexp.MustCompile("\\s*,\\s*").Split(strings.ReplaceAll(access," ",""), -1)
  for _,a := range s { 
    if (len(a)>0) {
      sq:="INSERT INTO "+tab+" VALUES(0,'"+a+"',"+strconv.Itoa(mode)+")"
      _, err := db.Exec(sq)
      if err != nil { 
        P(Crb(err.Error()))
        os.Exit(0)
      }
    }
  }
}

func getmypw() string { // ---------------------------------------------- try to get mysql password from .my.cnf
  homedir,_:=os.UserHomeDir()
  mycnf:=homedir+"/.my.cnf"
  readFile, err := os.Open(mycnf)
  if err == nil {
    fileScanner := bufio.NewScanner(readFile)
    fileScanner.Split(bufio.ScanLines)
    r1 := regexp.MustCompile("user=(.*)$")
    r2 := regexp.MustCompile("password=(.*)$")
    mf:=0
    for fileScanner.Scan() {
      line:=fileScanner.Text()
      ss := r1.FindStringSubmatch(line)
      if (len(ss)>0) {
        if ss[1]=="root" { mf=1 } else { mf=0 }
      }
      ss = r2.FindStringSubmatch(line)
      if (len(ss)>0 && mf==1) {
        return ss[1]
      }
    }
    readFile.Close()
  }
  return ""
}

// --------------------------------------------------------------------------------------------------------- END
