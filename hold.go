package main
import (
	"sort"
	"encoding/json"
	"time"
)
//持仓股票的最新交易记录
func getHoldLastDeal() (stocks []Stock) {
	//先尝试从redis中抓取持仓股票的最新交易记录，如果不成，再查数据库！
	key:="stock:hold:deal:recent"
	val, err := client.Get(ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&stocks)
		return
	} else {
		logger.Println("get hold stock deal list from redis error: ",err)
	}
	allHold:=getCodeList("hold")
	inSql:=`
		select date,code,name,operation,volume,balance,
		price,turnover,amount,brokerage,stamps,transfer_fee from stock 
		where code=?  and  operation in ("申购中签", "配股缴款","证券买入","红股入账","股息红利税补") order by date desc limit 1
	`
	outSql:=`
	select date,code,name,operation,volume,balance,
	price,turnover,amount,brokerage,stamps,transfer_fee from stock 
	where code=?  and  operation in ("证券卖出","股息入账") order by date desc limit 1
`
	for _,c:=range allHold {
		//先加买入股票类的操作，没有则不加
		s:=Stock{}
		err:=Db.QueryRow(inSql,c).Scan(
			&s.Date,&s.Code,&s.Name,&s.Operation,&s.Volume,&s.Balance,
			&s.Price,&s.Turnover,&s.Amount,&s.Brokerage,&s.Stamps,&s.TransferFee,
		)
		if err==nil {
			stocks=append(stocks,s)
		}
		//再加卖出股票类的操作，没有则不加
		s=Stock{}
		err=Db.QueryRow(outSql,c).Scan(
			&s.Date,&s.Code,&s.Name,&s.Operation,&s.Volume,&s.Balance,
			&s.Price,&s.Turnover,&s.Amount,&s.Brokerage,&s.Stamps,&s.TransferFee,
		)
		if err==nil {
			stocks=append(stocks,s)
		}
	}
	s,err:=json.Marshal(stocks)
    if err!=nil {
        logger.Println("set hold stock deal list serialize error: ",err)
    } else {
        client.Set(ctx, key, string(s), 75*time.Hour)
    }
	return
}
//持仓股票的成本分析
func getPositionStats()(holds Holds) {
	//先尝试从redis中抓取持仓股票的最新交易记录，如果不成，再查数据库！
	key:="stock:hold:stats"
	val, err := client.Get(ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&holds)
		return
	} else {
		logger.Println("get hold stock stats from redis error: ",err)
	}
	allHold:=getCodeList("hold")
	holdStocks:=getNameMap(allHold)
	var sql1,sql2,sql3 string
	sql1=`
		select sum(amount),min(date),max(date),datediff(curdate(),min(date)),
		sum(volume) from stock where code=?
	`
	sql2=`select count(id) from stock where code=? and operation in ('申购中签','证券买入','证券卖出')`
	sql3=`select date,balance from stock where code=? and balance=(select max(balance) from stock where code=?)`
	for k,v:=range holdStocks {
		hold:=Hold{}
		//第一步：先拿到总成本、首次及最近交易日期、持股天数、持仓量、计算平均成本
		Db.QueryRow(sql1,k).Scan(
			&hold.Amount,&hold.FirstDay,&hold.LastDay,&hold.HoldDays,&hold.Balance,
		)
		hold.AvgCost=float32(hold.Amount)/float32(hold.Balance)
		//第二步：获得买卖次数，并计算周买卖频率
		Db.QueryRow(sql2,k).Scan(&hold.TransactionCount)
		hold.TransactionFreq=float32(hold.TransactionCount)/float32(hold.HoldDays)*7
		//第三步，取出最高持仓量与相应日期
		Db.QueryRow(sql3,k,k).Scan(&hold.MaxBalanceDay,&hold.MaxBalance)
		hold.Code=k
		hold.Name=v
		holds.Costs+=hold.Amount
		holds.HoldList=append(holds.HoldList,hold)
	}
	sort.Sort(ByCost(holds.HoldList))
	s,err:=json.Marshal(holds)
    if err!=nil {
        logger.Println("set hold stock deal list serialize error: ",err)
    } else {
        client.Set(ctx, key, string(s), 75*time.Hour)
    }
	return
}