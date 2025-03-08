-- 清仓股票的代码列表
select distinct code from stock group by code having sum(volume)=0 order by code;
-- 持仓股票的代码列表
select distinct code from stock group by code having sum(volume)!=0 order by code;
-- 清仓股票中的打新部分
select code from stock where code in (select distinct code from stock group by code having sum(volume)=0) and operation='申购中签' order by code;

/*打新部分，可能还要细化，按以下的股票代码分类，分成5类，或者简单些，分成3类，也可以
*/
-- A股代码分类
/*
0开头，深市主板，通常是00开头
1开头，可转债，从目前的情况来看，11与12开头，均有
3开头，深市创业板，基本都是30开头
60开头，沪市主板
68开头，沪市科创板
8或9开头，北证，本人没操作过，后续也不会操作
*/
-- 清仓股票中的非新股部分
/* 短、中、长线判断标准
1. 新股肯定是要排除在外的，所以操作上，只有证券买入与证券卖出两种，没有申购中签！
2. 短线：持股时间一般在一周以内，但不超过一个月，也就是30天，算短线，这里是总时间，不是交易日数量，后面中长线，也是这样认定(<=30)
3. 中线：持股时间在1个月以上，半年以下(<=180)
4. 长线：持股时间在半年以上(>=180)
*/
--- 可转债只数查询
SELECT year(date) y ,count(id) FROM `stock` where operation in ('申购中签','配股缴款') and code regexp '^1' group by y;
-- 主板只数查询
SELECT year(date) y ,count(id) FROM `stock` where operation in ('申购中签','配股缴款') and code regexp '^[0,3,6,8,9]' group by y;

