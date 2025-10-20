package main

import (
	"time"
	"encoding/json"
	"fmt"
	"strings"
)
//完整的交易记录结构
type Stock struct {
	Id                               int
	Date      time.Time      //日期
	Code, Name, Operation     string //代码、名称、操作
	Volume, Balance     int    //数量、变动后持股数量
	Price, Turnover, Amount,Brokerage,Stamps,TransferFee    float32    //均价、成交金额、发生金额、佣金、印花税、过户费
}
// 持仓统计结构
type Stats struct {
	Base
	Senior
	Amount float32
}
//最基础统计字段
type Base struct {
	Code, Name                               string
	FirstDay, LastDay time.Time
	HoldDays int
}
type Senior struct {
	MaxBalanceDay time.Time
	MaxBalance, TransactionCount   int
	TransactionFreq float32
}
type Clear struct {
	Profit            float32
	ProfitRate        float32 //利润率
	ProfitPct         string  //以百分比显示的利润率
	AvgDailyProfit    float32
}
type NewShare struct {
	Base
	Clear
}
type NewShares struct {
	Profits          float32
	Kind, SortMethod string
	NewShareList     []NewShare
}

type Hold struct {
	Stats
	Balance int
	AvgCost float32
}
type Holds struct {
	Costs    float32
	HoldList []Hold
}

type NormClear struct {
	NewShare
	Senior
}
type NormClears struct {
	Profits            float32
	SortMethod, Period string
	NormClearList      []NormClear
}
//新版本的代码与名称映射，不依据查询条件，而是直接依赖代码列表来
func getNameMap(codes []string) map[string]string {
	//拿最新名字列表
	names:=make(map[string]string)
	sql:="select name from stock where code=? order by date desc"
	for _,c:=range codes {
		name:=""
		Db.QueryRow(sql,c).Scan(&name)
		names[c]=name
	}
	//logger.Println("codes name map: ", names)
	return names
}
//持仓成本排序方案保持不变
type ByCost []Hold
func (a ByCost) Len() int { return len(a) }
func (a ByCost) Less(i, j int) bool { return a[i].Amount < a[j].Amount }
func (a ByCost) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
//新股的三种排序方案
type ByProfitReverseNS []NewShare
func (a ByProfitReverseNS) Len() int { return len(a) }
func (a ByProfitReverseNS) Less(i, j int) bool { return a[i].Profit > a[j].Profit }
func (a ByProfitReverseNS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByProfitDailyReverseNS []NewShare
func (a ByProfitDailyReverseNS) Len() int { return len(a) }
func (a ByProfitDailyReverseNS) Less(i, j int) bool { return a[i].AvgDailyProfit > a[j].AvgDailyProfit }
func (a ByProfitDailyReverseNS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByProfitRateReverseNS []NewShare
func (a ByProfitRateReverseNS) Len() int { return len(a) }
func (a ByProfitRateReverseNS) Less(i, j int) bool { return a[i].ProfitRate > a[j].ProfitRate }
func (a ByProfitRateReverseNS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
//普通清仓股的七种排序方案
type ByProfitReverseCS []NormClear
func (a ByProfitReverseCS) Len() int { return len(a) }
func (a ByProfitReverseCS) Less(i, j int) bool { return a[i].Profit > a[j].Profit }
func (a ByProfitReverseCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByProfitDailyReverseCS []NormClear
func (a ByProfitDailyReverseCS) Len() int { return len(a) }
func (a ByProfitDailyReverseCS) Less(i, j int) bool { return a[i].AvgDailyProfit > a[j].AvgDailyProfit }
func (a ByProfitDailyReverseCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByProfitRateReverseCS []NormClear
func (a ByProfitRateReverseCS) Len() int { return len(a) }
func (a ByProfitRateReverseCS) Less(i, j int) bool { return a[i].ProfitRate > a[j].ProfitRate }
func (a ByProfitRateReverseCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }


type ByHoldDayCS []NormClear
func (a ByHoldDayCS) Len() int { return len(a) }
func (a ByHoldDayCS) Less(i, j int) bool { return a[i].HoldDays > a[j].HoldDays }
func (a ByHoldDayCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByMaxBalanceCS []NormClear
func (a ByMaxBalanceCS) Len() int { return len(a) }
func (a ByMaxBalanceCS) Less(i, j int) bool { return a[i].MaxBalance > a[j].MaxBalance }
func (a ByMaxBalanceCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByTransCountCS []NormClear
func (a ByTransCountCS) Len() int { return len(a) }
func (a ByTransCountCS) Less(i, j int) bool { return a[i].TransactionCount > a[j].TransactionCount }
func (a ByTransCountCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ByWeekFreqCS []NormClear
func (a ByWeekFreqCS) Len() int { return len(a) }
func (a ByWeekFreqCS) Less(i, j int) bool { return a[i].TransactionFreq > a[j].TransactionFreq }
func (a ByWeekFreqCS) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
//获取股票的代码列表，为了简单起见，不增加复杂的条件，仅分三种，清仓、持仓、全部三类(kind:  clear/hold/all)
func getCodeList(kind string) (codes []string) {
	//先尝试从redis中抓取清仓股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:%s:codes",kind)
	val, err := client.Get(ctx, key).Result()
	if err == nil {
        json.Unmarshal([]byte(val),&codes)
		return
	} else {
		errinfo:=fmt.Sprintf("get %s stock code list from redis error: %s",kind,err)
        logger.Println(errinfo)
    }
	var sql string
	//从数据库中查结果
	switch kind {
	case "clear":
		sql="select distinct code from stock group by code having sum(volume)=0 order by code"
	case "hold":
		sql="select distinct code from stock group by code having sum(volume)!=0 order by code"
	case "all":
		sql="select distinct code from stock order by code"
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
        client.Set(ctx, key, string(s), 75*time.Hour)
    }
    return
}

//以更好看的方式，显示全部的百分比数值
func perDisp(f float32) (fs string) {
	fs = fmt.Sprintf("%.2f", f)
	for {
		hasDot, TrailZero := strings.Contains(fs, "."), strings.HasSuffix(fs, "0")
		if !TrailZero || !hasDot {
			break
		} else {
			fs = strings.TrimSuffix(fs, "0")
			fs = strings.TrimSuffix(fs, ".")
		}
	}
	return
}