package main

import (
	"time"
	"encoding/json"
	"fmt"
)
//完整的交易记录结构
type Stock struct {
	Id                               int
	Date      time.Time      //日期
	Code, Name, Operation     string //代码、名称、操作
	Volume, Balance     int    //数量、变动后持股数量
	Price, Turnover, Amount,Brokerage,Stamps,TransferFee    float32    //均价、成交金额、发生金额、佣金、印花税、过户费
}
//公共统计结构
type Stats struct {
	Code,Name string
	FirstDealDay,LastDealDay,MaxBalanceDay time.Time
	HoldDays,MaxBalance,TransactionCount int
	//amount的总和，为正则是利润，为负则是成本
	Amount,TransactionFreq  float32
}
//依据可能的条件，获得代码与最新名称的映射
func getNameMap(cond string) map[string]string {
	//先拿代码列表
	sql:="select distinct code from stock "+cond
	codes:=[]string{}
	rows, _ := Db.Query(sql)
	for rows.Next() {
		c:=""
		rows.Scan(&c)
		codes = append(codes, c)
	}
	rows.Close()
	//再拿最新名字列表
	sql="select name from stock where code=? order by date desc"
	names:=make(map[string]string)
	for _,c:=range codes {
		name:=""
		Db.QueryRow(sql,c).Scan(&name)
		names[c]=name
	}
	return names
}
type ByProfitReverse []Clear
func (a ByProfitReverse) Len() int { return len(a) }
func (a ByProfitReverse) Less(i, j int) bool { return a[i].Amount > a[j].Amount }
func (a ByProfitReverse) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByCost []Hold
func (a ByCost) Len() int { return len(a) }
func (a ByCost) Less(i, j int) bool { return a[i].Amount < a[j].Amount }
func (a ByCost) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

//获取股票的代码列表，为了简单起见，不增加复杂的条件，仅分清仓与持仓两类(kind:  clear/hold)
func getCodeList(kind string) (codes []string) {
	//先尝试从redis中抓取清仓股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:%s:codes",kind)
	val, err := client.Get(key).Result()
	if err == nil {
        json.Unmarshal([]byte(val),&codes)
		return
	} else {
		errinfo:=fmt.Sprintf("get %s stock code list from redis error: %s",kind,err)
        logger.Println(errinfo)
    }
	var sql string
	//从数据库中查结果
	if kind=="clear" {
		sql="select distinct code from stock group by code having sum(volume)=0 order by code"
	} else if kind=="hold" {
		sql="select distinct code from stock group by code having sum(volume)!=0 order by code"
	}
	rows, _ := Db.Query(sql)
	for rows.Next() {
		c:=""
		rows.Scan(&c)
		codes = append(codes, c)
	}
	rows.Close()
	//写入redis，下次加速
	s,err:=json.Marshal(codes)
    if err!=nil {
		errinfo:=fmt.Sprintf("get %s stock code list serialize error: %s",kind,err)
        logger.Println(errinfo)
    } else {
        client.Set(key, string(s), 75*time.Hour)
    }
    return
}
