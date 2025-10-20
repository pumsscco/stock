package main
//已清仓的新股，包括可转债在内，简要分析
import (
	"sort"
	"time"
	"strings"
	"fmt"
	"encoding/json"
)
//打新类的代码列表，分可转债与主板两类
func getNewShareCodes(kind string) (codes []string) {
	//先尝试从redis中抓取打新股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:new-share:codes:%s",kind)
	val, err := client.Get(ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&codes)
		return
	} else {
		errinfo:=fmt.Sprintf("get clear stock code list of %s from redis error: %s",kind,err)
		logger.Println(errinfo)
	}
	allClear:=getCodeList("clear")
	for _, c := range allClear {
		//清仓股票再过滤，只有首次购买为申购中签或配股缴款是的，才是新股
		sql:=`select code,date,operation from stock group by code having code=? and date=min(date) and operation in ('申购中签', '配股缴款')`
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
        client.Set(ctx, key, string(s), 75*time.Hour)
    }
	return
}
//打新类的完整统计信息,种类分主板与可转债两种，排序方法分利润、日均利润、利润率三种
func getNewShareStats(kind,sortMethod string)(newShares NewShares) {
	//先尝试从redis中抓取打新股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:new-share:stats:sort:%s:%s",kind,sortMethod)
	val, err := client.Get(ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&newShares)
		return
	} else {
		errinfo:=fmt.Sprintf("get cleared new share stats list of %s sorting %s from redis error: %s",kind,sortMethod, err)
		logger.Println(errinfo)
	}
	//先依类别，抓代码列表，再抓最新的名称与代码的映射
	newShareCodes:=getNewShareCodes(kind)
	//logger.Println("newShare codes: ", newShareCodes)
	newShareMaps:=getNameMap(newShareCodes)
	var sql1,sql2 string
	sql1=`
		select sum(amount),min(date),max(date),datediff(max(date),min(date)),
		sum(amount)/datediff(max(date),min(date)) from stock where code=?
	`
	if kind=="cb" {
		sql2=`select turnover from stock where code=? and operation in ('申购中签', '配股缴款')`
	} else if kind=="main" {
		sql2=`select turnover from stock where code=? and operation='申购中签'`
	}
	//然后统计数据
	for k,v:=range newShareMaps {
		newShare:=NewShare{}
		//第一步：先拿到净利润、首末交易日期、持股天数、日均利润
		Db.QueryRow(sql1,k).Scan(
			&newShare.Profit,
			&newShare.FirstDay,
			&newShare.LastDay,
			&newShare.HoldDays,
			&newShare.AvgDailyProfit,
		)
		//第二步：计算利润率，新股的利润比较好算，只要利润除以申购的成本就行了，新股申购没有佣金等额外成本，就取个巧
		var cost float32
		Db.QueryRow(sql2,k).Scan(&cost)
		newShare.ProfitRate = newShare.Profit / cost 
		newShare.ProfitPct = fmt.Sprintf("%s%%", perDisp(float32(newShare.ProfitRate*100)))
		newShare.Code = k
		newShare.Name = v
		newShares.Profits += newShare.Profit
		newShares.NewShareList=append(newShares.NewShareList,newShare)
	}
	switch sortMethod {
	case "profit":
		newShares.SortMethod="净利润"
		sort.Sort(ByProfitReverseNS(newShares.NewShareList))
	case "profit-daily":
		newShares.SortMethod="日均利润"
		sort.Sort(ByProfitDailyReverseNS(newShares.NewShareList))
	case "profit-rate":
		newShares.SortMethod="利润率"
		sort.Sort(ByProfitRateReverseNS(newShares.NewShareList))
	}
	switch kind {
	case "cb":
		newShares.Kind="可转债"
	case "main":
		newShares.Kind="主板"
	}
	s,err:=json.Marshal(newShares)
    if err!=nil {
		errinfo:=fmt.Sprintf("set cleared new share stats list of %s sorting %s serialize error: %s",kind,sortMethod, err)
        logger.Println(errinfo)
    } else {
        client.Set(ctx, key, string(s), 75*time.Hour)
    }
	return
}
