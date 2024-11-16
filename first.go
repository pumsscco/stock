package main
//已清仓的新股，包括可转债在内，简要分析
import (
	"sort"
	"time"
	"strings"
	"fmt"
	"encoding/json"
)
type NewShare struct {
	Code,Name string
	FirstDay,LastDay time.Time
	HoldDays int
	//amount的总和，为正则是利润，为负则是成本
	Profit  float32
	ProfitPer  float32  //利润率
	AvgDailyProfit float32
}
type NewShares struct {
	Profits float32
	NewShareList []NewShare
}
//打新类的代码列表，分可转债与主板两类
func NewShareCodes(kind string) (codes []string) {
	//先尝试从redis中抓取打新股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:clear:codes:%s",kind)
	val, err := client.Get(key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&codes)
		return
	} else {
		errinfo:=fmt.Sprintf("get clear stock code list of %s from redis error: %s",kind,err)
		logger.Println(errinfo)
	}
	allClear:=getCodeList("clear")
	for _, c := range allClear {
		//清仓股票再过滤，只有首次购买为申购中签的，才是新股
		sql:=`select code,date,operation from stock group by code having code=? and date=min(date) and operation='申购中签'`
		var code,date,operation string
		err=Db.QueryRow(sql,c).Scan(&code,&date,&operation)
		//只有确认了新股身份，才执行是主板，还是可转债的判断
		if err==nil && code!="" {
			if kind=="cb" && strings.HasPrefix(code, "1") {
				codes=append(codes, code)
			} else if kind=="main" && !strings.HasPrefix(c,"1") {
				codes=append(codes, code)
			}
		}
	}
	s,err:=json.Marshal(codes)
    if err!=nil {
		errinfo:=fmt.Sprintf("get clear stock code list of %s serialize error: %s",kind,err)
        logger.Println(errinfo)
    } else {
        client.Set(key, string(s), 75*time.Hour)
    }
	return
}
//打新类的完整统计信息
func getNewShareStats(kind,sortMethod string)(newShares NewShares) {
	//先尝试从redis中抓取打新股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:clear:stats:sort:%s:%s",kind,sortMethod)
	val, err := client.Get(key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&newShares)
		return
	} else {
		errinfo:=fmt.Sprintf("get clear stock stats list of %s from redis error: %s",kind,err)
		logger.Println(errinfo)
	}
	//先依类别，抓代码列表，再抓最新的名称与代码的映射
	newShareCodes:=NewShareCodes(kind)
	newShareMaps:=getNameMapNew(newShareCodes)
	//然后统计数据
	for k,v:=range newShareMaps {
		newShare:=NewShare{}
		//第一步：先拿到净利润、首末交易日期、持股天数、日均利润
		sql:=`
			select sum(amount),min(date),max(date),datediff(max(date),min(date)),
			sum(amount)/datediff(max(date),min(date)) from stock where code=?
		`
		Db.QueryRow(sql,k).Scan(
			&newShare.Profit,
			&newShare.FirstDay,
			&newShare.LastDay,
			&newShare.HoldDays,
			&newShare.AvgDailyProfit,
		)
		//第二步：计算利润率，新股的利润比较好算，只要利润除以申购的成本就行了，新股申购没有佣金等额外成本，就取个巧
		sql=`select turnover from stock where code=? and operation='申购中签'`
		var cost float32
		Db.QueryRow(sql,k).Scan(&cost)
		newShare.ProfitPer=(newShare.Profit-cost)/cost*100%
		newShare.Code=k
		newShare.Name=v
		newShares.Profits+=newShare.Profit
		newShares.NewShareList=append(newShares.NewShareList,newShare)
	}
	sort.Sort(ByProfitReverseNS(newShares.NewShareList))
	s,err:=json.Marshal(codes)
    if err!=nil {
		errinfo:=fmt.Sprintf("get clear stock code list of %s serialize error: %s",kind,err)
        logger.Println(errinfo)
    } else {
        client.Set(key, string(s), 75*time.Hour)
    }
	return
}
