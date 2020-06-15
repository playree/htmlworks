# HTML Works
`HTML Works` は、テンプレートエンジンを使用できる静的HTMLの開発ツールです。  
各種フレームワークを使用したWEBプログラム開発ではよく使用されるHTMLのテンプレートエンジンですが、
そのテンプレートエンジンを静的HTMLの開発でも使用できるようにしたのがこのツールになります。  
同種のものに [`HUGO`](https://gohugo.io/) というツールなどもあるのですが、
こちらはコンテンツ部がMarkdown記述だったりと少々自分のニーズに合わなかった為、独自に開発してみました。

ひとまず動くものとなったので公開していますが、絶賛開発途中となります。  
機能追加や改善などの要望がございましたら、[GitHub Issues](https://github.com/playree/goingtpl/issues) からご連絡ください。  
対応のお約束はできませんが、今後の開発の参考にさせていただきます。

主な特徴として、

* HTMLのテンプレートエンジンを使用できる  
テンプレートエンジンには、Go言語の `html/template` を採用しています。(というかそのままですｗ)  
テンプレート記述のみで完結できるよう、テンプレートエンジンを多少拡張しています。

* WEBサイト公開用の静的HTMLを生成できる  
テンプレートエンジンを使用した記述方法で作成したコンテンツから、公開用の静的HTMLを作成します。  
本ツールの主機能になります。

* 開発用の簡易サーバーを使用して静的HTML生成前にコンテンツを確認できる  
開発用の簡易サーバー機能を用意していますので、編集中のコンテンツをリアルタイムで確認できます。

* Windows/Mac/Linuxの各種環境で使用可能  
Go言語で作成していますので、ビルドさえ行えば各種環境で使用可能です。  
バイナリ(実行形式)での提供は徐々に増やす予定です。

# Table of content
* [Getting Started](#Getting%20Started)
* [Usage](#Usage)
* [Template Engine](#Template%20Engine)
* [License](#License)
* [History](#History)

# Getting Started
下記GitHubから最新版をダウンロードしてください。  
[https://github.com/playree/htmlworks/releases](https://github.com/playree/htmlworks/releases)

ダウンロードしたファイルを解凍すると、`bin`フォルダ配下に各種バイナリがありますので、
環境変数に追加するなどしてパスを通してください。  
大変申し訳ありませんが、ご自身の環境向けのバイナリが存在しない場合はビルドしてお使いください。

ビルドコマンド
```
go build htmlworks.go
```

# Usage

## 開発環境の初期化
まずは `HTML Works` で開発する為の設定ファイルやディレクトリを作成します。  
環境を構築したいディレクトリ配下で下記コマンドを実行してください。
```
htmlworks init
```

下記が生成されます。

* htmlworks.toml (設定ファイル)
* contents (ディレクトリ)
* resources (ディレクトリ)

## コンテンツの作成
`contents`ディレクトリ配下にHTMLを作成していきます。  
例えば下記のように。

* contents/index.html  
    ```
    <!--params
        {
            "title":"HTML Works",
    
        }
    -->
    {{extends "_parts/base.html"}}
    {{define "content"}}
    <p>Test Contents</p>
    {{end}}
    ```

* contents/_parts/base.html  
    ```
    <html>
    <head>
        <title>{{.title}}</title>
        <link rel="stylesheet" href="/resources/css/kuroutokit.css" />
    </head>
    <body>
        {{template "content" .}}
    </body>
    </html>
    ```

リソースなどは`resources`ディレクトリに配置します。

* resources/css/kuroutokit.css

コンテンツを配置したら下記コマンドで静的HTMLを生成します。
```
htmlworks gen
```

`public`ディレクトリに下記のように生成されます。

* public/index.html  
    ⇒contents/index.html と contents/_parts/base.html から下記内容で生成されます。  
    　そして、contents/_partsディレクトリ配下は生成対象になりません。  
    ```
    <html>
    <head>
        <title>HTML Works</title>
        <link rel="stylesheet" href="/resources/css/kuroutokit.css" />
    </head>
    <body>
        <p>Test Contents</p>
    </body>
    </html>
    ```

* public/resources/css/kuroutokit.css  
    ⇒resources/css/kuroutokit.css からコピーされます。

## 簡易サーバーで確認する
下記コマンドで開発確認用の簡易サーバーが立ち上がります。

```
htmlworks serve
```
```
2020/06/14 16:28:11 HTML Works ver 1.0.0
2020/06/14 16:28:11 args: serve
2020/06/14 16:28:11 Load Setting > ./htmlworks.toml
2020/06/14 16:28:11 Start Server ========
2020/06/14 16:28:11 Port: 8088
2020/06/14 16:28:11 Contents directory: contents
2020/06/14 16:28:11 Resources directory: resources
2020/06/14 16:28:11 Starting development server at http://localhost:8088/
2020/06/14 16:28:11 Quit the server with CONTROL-C.
```

`http://localhost:8088/` にアクセスすることで編集中のコンテンツをリアルタイムで確認できます。  
この場合は、`contents/index.html` の静的HTML化した際の内容が表示されます。  
簡易サーバーで確認する際には、静的HTMLを作成しておく必要はありません。(サーバーが動的に静的HTMLした結果を表示します)

# Template Engine

基本的にはGo言語の `html/template` 仕様に従います。

さらに `HTML Works` として拡張した仕様が下記になります。

## パラメータ定義
下記のようにパラメータを定義して使用することができます。  
※パラメータ定義 `<!--params` ～ `-->` を記述する場合は、必ずファイルの先頭に記述してください。

* contents/index.html  
    ```
    <!--params
    {
        "str1":"abc",
        "int1":123,
        "bool1":true,
        "array1":["a1","a2","a3"]
    }
    -->
    <html>
    <body>
        <p>str1:{{.str1}}</p>
        <p>int1:{{.int1}}</p>
        <p>
        bool1:{{.bool1}}
        {{if .bool1}}
            <p>if TRUE</p>
        {{end}}
        </p>
        <p>
        array1:
        <ul>
            {{range $i, $val := .array1}}
            <li>{{$i}} : {{$val}}</li>
            {{end}}
        </ul>
        </p>
    </body>
    </html>
    ```

上記から生成される静的HTMLは下記になります。

* public/index.html  
    ```
    <html>
    <body>
        <p>str1:abc</p>
        <p>int1:123</p>
        <p>
        bool1:true
            <p>if TRUE</p>
        </p>
        <p>
        array1:
        <ul>
            <li>0 : a1</li>
            <li>1 : a2</li>
            <li>2 : a3</li>
        </ul>
        </p>
    </body>
    </html>
    ```

## 継承(extends)
下記 `{{extends "_parts/xxx.html"}}` のように記述することで、`xxx.html` をベースに一部分を差し替えたコンテンツを作成することができます。  
`{{template "xxx" .}}` の部分を `{{define "xxx"}}` ～ `{{end}}` の内容で差し変えることができます。  
継承元でもパラメータ定義は使用することができます。  
※ただし、パラメータ定義は継承元に記述することはできません。

* contents/index.html (継承先)  
    ```
    <!--params
    {
        "title":"HTML Works"
    }
    -->
    {{extends "_parts/base.html"}}
    {{define "content"}}
    <p>Extends Contents</p>
    {{end}}
    ```

* contents/_parts/base.html (継承元)  
    ```
    <html>
    <head>
        <title>{{.title}}</title>
        <link rel="stylesheet" href="/resources/css/kuroutokit.css" />
    </head>
    <body>
        {{template "content" .}}
    </body>
    </html>
    ```

上記から生成される静的HTMLは下記になります。

* public/index.html  
    ```
    <html>
    <head>
        <title>HTML Works</title>
        <link rel="stylesheet" href="/resources/css/kuroutokit.css" />
    </head>
    <body>
        <p>Extends Contents</p>
    </body>
    </html>
    ```

## インクルード(Include)
下記 `{{include "_parts/xxx.html"}}` のように記述することで、`xxx.html` の内容を取り込むことができます。  
`{{template "xxx" .}}` の部分が `{{define "xxx"}}` ～ `{{end}}` の内容に差し変わります。

* contents/index.html (インクルード先)  
    ```
    <html>
    <head>
        {{template "head" .}}{{include "_parts/head.html"}}
    </head>
    <body>
        <p>Include Test</p>
    </body>
    </html>
    ```

* contents/_parts/head.html (インクルード元)  
    ```
    {{define "head"}}
        <link rel="stylesheet" href="/resources/css/kuroutokit.css" />
    {{end}}
    ```

上記から生成される静的HTMLは下記になります。

* public/index.html  
    ```
    <html>
    <head>
        <link rel="stylesheet" href="/resources/css/kuroutokit.css" />
    </head>
    <body>
        <p>Include Test</p>
    </body>
    </html>
    ```

## 組み込み関数
いくつかの組み込み関数を使用することができます。  
組み込み関数については、今後徐々に追加を予定しています。

* now  
    引数 : フォーマット  
    現在時刻を使用できます。  
    フォーマットを省略すると `2006/01/02 15:04:05` が使用されます。
    ```
    <p>最終更新 : {{now "2006年01月02日 15:04:05"}}</p>
    <p>Last Update : {{now ""}}</p>
    ```
    生成後は下記のようになります。
    ```
    <p>最終更新 : 2020年06月14日 19:23:53</p>
    <p>Last Update : 2020/06/14 19:23:53</p>
    ```
    更新日時の自動挿入などに利用できます。

# License
[MIT](https://github.com/playree/htmlworks/blob/master/LICENSE)

Copyright (c) 2020 Kazuki Minakawa (funlab, Inc. https://funlab.jp)

# History

| Date       | Ver   | Content |
| :--------- | :---- | :------ |
| 2020/06/15 | 0.1.0 | リリース。 |