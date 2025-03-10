package main

import (
    "fmt"
    "database/sql"
    "log"
    "net/http"
    yaml "gopkg.in/yaml.v2"
	"github.com/go-redis/redis"
	"io/ioutil"
    "github.com/julienschmidt/httprouter"
    _ "github.com/go-sql-driver/mysql"
    "os"
)
var Db *sql.DB
var logger *log.Logger
var client *redis.Client
type Conf struct {
    Listen struct {
        Host string `yaml:"host"`
        Port int `yaml:"port"`
    }
    MySQL struct {
        Db string `yaml:"db"`
        Host string `yaml:"host"`
        Port int `yaml:"port"`
        User string `yaml:"user"`
        Pass string `yaml:"pass"`
    }
    Redis struct {
        Host string `yaml:"host"`
        Port int `yaml:"port"`
        Db int `yaml:"db"`
        Pass string `yaml:"pass"`
    }
	Logfile string `yaml:"logfile"`
}
var cnf Conf
func init() {
    //抓全部的配置信息
    yamlBytes, _ := ioutil.ReadFile("config.yml")
    yaml.Unmarshal(yamlBytes,&cnf)
    file, err := os.OpenFile(cnf.Logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalln("无法打开日志文件", err)
    }
    logger = log.New(file, "INFO ", log.Ldate|log.Ltime|log.Lshortfile)
    client=redis.NewClient(&redis.Options{
        Addr:       fmt.Sprintf("%s:%d",cnf.Redis.Host,cnf.Redis.Port),
        Password:   cnf.Redis.Pass,
        DB:         cnf.Redis.Db,
    })
    _, err = client.Ping().Result()
    if err!=nil {
        logger.Fatalf("redis连接异常：%v\n",err)
    }  
    dsn:=fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?loc=Local&parseTime=true", cnf.MySQL.User, cnf.MySQL.Pass, cnf.MySQL.Host, cnf.MySQL.Port, cnf.MySQL.Db)
    //fmt.Println("Data Source Name: ",dsn)
    Db,err=sql.Open("mysql",dsn)
    if err!=nil {
        logger.Fatalf("open mysql failed: %v",err)
    }
}
func main() {
    // handle static assets
    router := httprouter.New()
    router.ServeFiles("/static/*filepath", http.Dir("static"))
    //页面
    router.GET("/", index)
    //以时间逆序列出该股票代码的相关所有交易记录，新增记录成功后，也会重定向到这里
    router.GET("/ipo/lot",ipoLot) //简单的打新数量统计
    router.GET("/ns/:Type", newStock) //打新类型简单些分类，就主板（main)与可转债（cb）两类
    router.POST("/ns/:Type", newStock) //选择好排序方式后，再给出最终结论
    router.GET("/cs/:Type", normalStock) //普通清仓股
    router.POST("/cs/:Type", normalStock) //同样要排序，复杂的多
    router.GET("/hs/operation", holdLastDeal) //持仓股的最新交易
    router.GET("/hs/cost", position) // 持仓股的成本分析
    router.GET("/other/single", dealList)  //获得股票名称列表
    router.POST("/other/single", dealList) //依据单一代码，获取其全部交易明细
    
    router.GET("/other/add", newDeal)   //新增记录表单
    router.POST("/other/add", newDeal)  //新记录入库

    log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d",cnf.Listen.Host,cnf.Listen.Port),router))
}
