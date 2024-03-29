package doubangroup

import (
	"github.com/Nrich-sunny/crawler/collect"
)

var DoubangroupJSTask = &collect.TaskModule{
	Property: collect.Property{
		Name:     "js_find_douban_sun_room",
		WaitTime: 2,
		MaxDepth: 5,
		Cookie:   "ll=\"118227\"; bid=ZYEsSfhim-4; viewed=\"1017143\"; gr_user_id=ab178b0e-82de-47c4-99fe-5221b995da91; __utmc=30149280; frodotk_db=\"ad691528e65057fd083b3f7ab2f107f7\"; __utmv=30149280.24935; dbcl2=\"249356040:nrPxmf/90qY\"; ck=WuR-; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1689773168%2C%22https%3A%2F%2Faccounts.douban.com%2F%22%5D; _pk_id.100001.8cb4=56f26c8ed8708b6e.1669633904.; push_noty_num=0; push_doumail_num=0; __utma=30149280.1232258161.1670930004.1670930004.1689773169.2; __utmz=30149280.1689773169.2.1.utmcsr=accounts.douban.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __yadk_uid=h928WDA73zU4Fnb7ttuaHTF5hnvU8yCl; douban-fav-remind=1; __gads=ID=b974d4f0d099827e-220633badad8009d:T=1670930004:RT=1689773653:S=ALNI_MZxUNCxXhloIXubkbdvrrwt2gWOUQ; __gpi=UID=00000b8f534707d0:T=1670930004:RT=1689773653:S=ALNI_MZ52YkdO2Wc_9jiLidVNsCOIplaWg",
	},
	Root: `
		var arr = new Array();
 		for (var i = 25; i <= 25; i+=25) {
			var obj = {
			   Url: "https://www.douban.com/group/szsh/discussion?start=" + i,
			   Priority: 1,
			   RuleName: "解析网站URL",
			   Method: "GET",
		   };
			arr.push(obj);
		};
		console.log(arr[0].Url);
		AddJsReq(arr);
		`,
	Rules: []collect.RuleModule{
		{
			Name: "解析网站URL",
			ParseFunc: `
				ctx.ParseJSReg("解析阳台房","(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>");
			`,
		},
		{
			Name: "解析阳台房",
			ParseFunc: `
				ctx.OutputJS("<div class="topic-content">[\s\S]*?阳台[\s\S]*?<div");
			`,
		},
	},
}
