# 前台標籤語法手冊 (TagParser Guide)

Gbootcms 前台視圖採用自主研發的高效能標籤引擎 `TagParser`，具備正則全局預編譯、樹狀 CTE 欄目遞歸與記憶體頁面快取能力。

---

## 1. 站點與全域變量

```html
<!-- 站點基本資訊 -->
<title>{gboot:site:title} - {gboot:site:subtitle}</title>
<meta name="keywords" content="{gboot:site:keywords}">
<meta name="description" content="{gboot:site:description}">
<link rel="icon" href="/{gboot:site:logo}">

<!-- 公司資訊 -->
<p>公司名稱：{gboot:company:name}</p>
<p>聯繫電話：{gboot:company:phone}</p>
<p>聯繫地址：{gboot:company:address}</p>
```

---

## 2. 導航選單 (`{gboot:nav}`)

```html
<ul class="nav">
    <li class="{if('{gboot:sort:scode}'=='')}active{/if}"><a href="/">首頁</a></li>
    {gboot:nav parent=0}
    <li class="{if('[nav:scode]'=='{gboot:sort:tcode}')}active{/if}">
        <a href="[nav:link]">[nav:name]</a>
        <!-- 二級下拉子選單 -->
        {if([nav:soncount]>0)}
        <ul class="sub-nav">
            {gboot:nav parent=[nav:scode]}
            <li><a href="[nav:link]">[nav:name]</a></li>
            {/gboot:nav}
        </ul>
        {/if}
    </li>
    {/gboot:nav}
</ul>
```

---

## 3. 文章/內容列表 (`{gboot:list}`)

支援分頁、多條件排序、推薦置頂篩選：

```html
<!-- 查詢欄目 scode=2 下的最新 10 篇文章 -->
{gboot:list scode=2 num=10 order='date desc'}
<div class="news-item">
    <a href="[list:link]">
        <img src="[list:ico]" alt="[list:title]">
        <h3>[list:title]</h3>
        <p>[list:description]</p>
        <span class="date">[list:date style='Y-m-d']</span>
    </a>
</div>
{/gboot:list}

<!-- 分頁條渲染 -->
<div class="pagination">
    {gboot:pagebar}
</div>
```

---

## 4. 內容詳情頁標籤

在詳情頁（如 `content.html`）中直接提取當前文章欄位：

```html
<article class="article-detail">
    <h1>{gboot:content:title}</h1>
    <div class="meta">
        <span>作者：{gboot:content:author}</span>
        <span>時間：{gboot:content:date style='Y-m-d H:i'}</span>
        <span>瀏覽次數：{gboot:content:visits}</span>
    </div>
    <div class="content-body">
        {gboot:content:content}
    </div>
    
    <!-- 上一篇 / 下一篇導航 -->
    <div class="prev-next">
        <p>上一篇：{gboot:content:pre}</p>
        <p>下一篇：{gboot:content:next}</p>
    </div>
</article>
```

---

## 5. 多語言字典替換 (`{i18n:...}`)

在單套 HTML 模板架構下，靜態 UI 文案使用 `{i18n:Key}` 標籤，系統依當前 URL 語言前綴動態注入翻譯：

```html
<button type="submit" class="btn-submit">{i18n:Submit}</button>
<input type="text" placeholder="{i18n:SearchPlaceholder}">
<a href="/member/login.html">{i18n:Login}</a>
```
