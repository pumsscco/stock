package main
//已清仓的普通股
import (
	"sort"
	"time"
	"fmt"
	"encoding/json"
)
//普通清仓股票代码列表, 按短中长来分（kind: short/medium/long)
func getNormClearCodes(kind string) (codes []string) {
	//先尝试从redis中抓取打新股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:norm-clear:codes:%s",kind)
	val, err := client.Get(key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&codes)
		return
	} else {
		errinfo:=fmt.Sprintf("get normal clear stock code list of %s from redis error: %s",kind,err)
		logger.Println(errinfo)
	}
	allClear:=getCodeList("clear")
	var sql string
	switch kind {
	case "short":
		sql=`select code,date,operation from stock group by code having code=? and date=min(date) and operation='证券买入' and  datediff(max(date),min(date))<30 `
	case "medium":
		sql=`select code,date,operation from stock group by code having code=? and date=min(date) and operation='证券买入' and  datediff(max(date),min(date)) between 30  and 180 `
	case "long":
		sql=`select code,date,operation from stock group by code having code=? and date=min(date) and operation='证券买入' and  datediff(max(date),min(date))>180 `
	}
	for _, c := range allClear {
		//清仓股票再过滤，首次购买证券买入，才是普通股
		var code,date,operation string
		err=Db.QueryRow(sql,c).Scan(&code,&date,&operation)
		if err==nil && code!="" {
			codes=append(codes, code)
		}
	}
	s,err:=json.Marshal(codes)
    if err!=nil {
		errinfo:=fmt.Sprintf("get normal clear stock code list of %s serialize error: %s",kind,err)
        logger.Println(errinfo)
    } else {
        client.Set(key, string(s), 75*time.Hour)
    }
	return
}
//普通清仓类的完整统计信息,种类分短、中、长期三种，排序方法分利润、日均利润、利润率、、、、、、七种
func getNormClearStats(kind,sortMethod string)(normClears NormClears) {
	//先尝试从redis中抓取打新股票的代码列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:norm-clear:stats:sort:%s:%s",kind,sortMethod)
	val, err := client.Get(key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&normClears)
		return
	} else {
		errinfo:=fmt.Sprintf("get cleared new share stats list of %s sorting %s from redis error: %s",kind,sortMethod, err)
		logger.Println(errinfo)
	}
	//先依类别，抓代码列表，再抓最新的名称与代码的映射
	normClearCodes:=getNormClearCodes(kind)
	normClearMaps:=getNameMap(normClearCodes)
	var sql1,sql2,sql3,sql4 string
	sql1=`
		select sum(amount),min(date),max(date),datediff(max(date),min(date)),
		sum(amount)/datediff(max(date),min(date)) from stock where code=?
	`
	sql2=`select count(id) from stock where code=? and operation in ('申购中签','证券买入','证券卖出')`
	sql3=`select date,balance from stock where code=? and balance=(select max(balance) from stock where code=?)`
	sql4=`select sum(amount) from stock where code=? and date<=?`
	//然后统计数据
	for k,v:=range normClearMaps {
		normClear:=NormClear{}
		//第一步：先拿到净利润、首末交易日期、持股天数、日均利润
		Db.QueryRow(sql1,k).Scan(
			&normClear.Profit,
			&normClear.FirstDay,
			&normClear.LastDay,
			&normClear.HoldDays,
			&normClear.AvgDailyProfit,
		)	
		//第二步：获得买卖次数，并计算周买卖频率
		Db.QueryRow(sql2,k).Scan(&normClear.TransactionCount)
		normClear.TransactionFreq=float32(normClear.TransactionCount)/float32(normClear.HoldDays)*7
		//第三步，取出最高持仓量与相应日期
		Db.QueryRow(sql3,k,k).Scan(&normClear.MaxBalanceDay,&normClear.MaxBalance)
		/*第四步：计算利润率，普通股的最大成本，很难统计，
		理论上，应该是以每次买入为计算节点，计算当前的最新总成本，然后全部对比，取出最高的，
		但这样太繁琐了，所以本人简单点，把最高持仓量所在的日期作为计算节点，
		只计算此时的总成本
		*/
		var cost float32
		Db.QueryRow(sql4,k,normClear.MaxBalanceDay).Scan(&cost)
		normClear.ProfitRate = -normClear.Profit / cost 
		normClear.ProfitPct = fmt.Sprintf("%s%%", perDisp(float32(normClear.ProfitRate*100)))
		normClear.Code = k
		normClear.Name = v
		normClears.Profits += normClear.Profit
		normClears.NormClearList=append(normClears.NormClearList,normClear)
	}
	switch sortMethod {
	case "profit":
		normClears.SortMethod="净利润"
		sort.Sort(ByProfitReverseCS(normClears.NormClearList))
	case "profit-daily":
		normClears.SortMethod="日均利润"
		sort.Sort(ByProfitDailyReverseCS(normClears.NormClearList))
	case "profit-rate":
		normClears.SortMethod="利润率"
		sort.Sort(ByProfitRateReverseCS(normClears.NormClearList))
	case "hold-day":
		normClears.SortMethod="持股天数"
		sort.Sort(ByHoldDayCS(normClears.NormClearList))
	case "max-balance":
		normClears.SortMethod="最高持仓量"
		sort.Sort(ByMaxBalanceCS(normClears.NormClearList))
	case "t-count":
		normClears.SortMethod="买卖次数"
		sort.Sort(ByTransCountCS(normClears.NormClearList))
	case "week-freq":
		normClears.SortMethod="周买卖频率"
		sort.Sort(ByWeekFreqCS(normClears.NormClearList))
	}
	switch kind {
	case "short":
		normClears.Period="短线操作"
	case "medium":
		normClears.Period="中线操作"
	case "long":
		normClears.Period="长线操作"
	}
	s,err:=json.Marshal(normClears)
    if err!=nil {
		errinfo:=fmt.Sprintf("set cleared normal stats list of %s sorting %s serialize error: %s",kind,sortMethod, err)
        logger.Println(errinfo)
    } else {
        client.Set(key, string(s), 75*time.Hour)
    }
	return
}
