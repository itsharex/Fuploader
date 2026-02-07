# B站视频发布接口文档

> 来源：https://github.com/biliup/biliup

---

## 目录

1. [登录验证接口](#1-登录验证接口)
2. [封面上传接口](#2-封面上传接口)
3. [预上传接口](#3-预上传接口)
4. [视频分片上传接口](#4-视频分片上传接口)
5. [分片确认接口](#5-分片确认接口)
6. [视频发布/提交接口](#6-视频发布提交接口)

---

## 1. 登录验证接口

验证 Cookie 是否有效，获取用户信息。

| 项目 | 内容 |
|------|------|
| **URL** | `https://api.bilibili.com/x/web-interface/nav` |
| **方法** | GET |
| **Headers** | `user-agent`, `cookie`, `Referer` |
| **用途** | 验证Cookie是否有效，获取用户信息 |

### 请求示例

```http
GET https://api.bilibili.com/x/web-interface/nav
Cookie: SESSDATA=xxx; bili_jct=xxx
Referer: https://www.bilibili.com
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36
```

### 响应示例

```json
{
  "code": 0,
  "message": "0",
  "ttl": 1,
  "data": {
    "isLogin": true,
    "face": "头像URL",
    "uname": "用户名",
    "mid": 12345678
  }
}
```

---

## 2. 封面上传接口

上传视频封面图片。

| 项目 | 内容 |
|------|------|
| **URL** | `https://member.bilibili.com/x/vu/web/cover/up` |
| **方法** | POST |
| **Content-Type** | `application/x-www-form-urlencoded` |
| **参数** | `cover` (base64编码图片), `csrf` |
| **用途** | 上传视频封面 |

### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| cover | string | 是 | Base64编码的图片数据 |
| csrf | string | 是 | CSRF Token（从Cookie中获取） |

### 请求示例

```http
POST https://member.bilibili.com/x/vu/web/cover/up
Content-Type: application/x-www-form-urlencoded
Cookie: SESSDATA=xxx; bili_jct=xxx
Referer: https://member.bilibili.com/video/upload.html

cover=data:image/jpeg;base64,/9j/4AAQSkZJRgABAQ...&csrf=xxx
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "url": "https://i0.hdslb.com/bfs/archive/xxx.jpg"
  }
}
```

---

## 3. 预上传接口

获取视频上传的配置信息，包括上传服务器地址、分片大小等。

| 项目 | 内容 |
|------|------|
| **URL** | `https://member.bilibili.com/preupload` |
| **方法** | GET |
| **Query参数** | 见下表 |
| **用途** | 获取上传配置信息（endpoint、chunk_size、auth等） |

### Query 参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| probe_version | string | 是 | 版本号，如 `20211012` |
| upcdn | string | 是 | CDN节点，如 `bda2` |
| zone | string | 是 | 区域，如 `cs` |
| name | string | 是 | 视频文件名 |
| r | string | 是 | 固定值 `upos` |
| profile | string | 是 | 固定值 `ugcfx/bup` |
| ssl | string | 是 | 是否使用SSL，`0`或`1` |
| version | string | 是 | 版本号，如 `2.10.4.0` |
| build | string | 是 | 构建号，如 `2100400` |
| size | string | 是 | 视频文件大小（字节） |
| webVersion | string | 是 | Web版本，如 `2.0.0` |

### 请求示例

```http
GET https://member.bilibili.com/preupload?probe_version=20211012&upcdn=bda2&zone=cs&name=video.mp4&r=upos&profile=ugcfx/bup&ssl=0&version=2.10.4.0&build=2100400&size=104857600&webVersion=2.0.0
Cookie: SESSDATA=xxx; bili_jct=xxx
Referer: https://member.bilibili.com/video/upload.html
```

### 响应示例

```json
{
  "OK": 1,
  "auth": "xxx",
  "endpoint": "upos-sz-mirrorcos.bilivideo.com",
  "upos_uri": "ugcfx2lf/n22010101xxx",
  "chunk_size": 4194304,
  "threads": 3,
  "biz_id": 1234567890,
  "upload_id": "xxx"
}
```

### 响应字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| auth | string | 上传认证令牌 |
| endpoint | string | 上传服务器地址 |
| upos_uri | string | 上传URI |
| chunk_size | int | 分片大小（字节） |
| threads | int | 并发线程数 |
| biz_id | int | 业务ID |
| upload_id | string | 上传会话ID |

---

## 4. 视频分片上传接口

将视频文件分片上传到 UPOS 服务器。

| 项目 | 内容 |
|------|------|
| **URL** | `https://{endpoint}/{upos_uri}`（从预上传接口返回） |
| **方法** | PUT |
| **Headers** | `Content-Type: application/octet-stream`, `X-Upos-Auth` |
| **Query参数** | 见下表 |
| **用途** | 上传视频分片 |

### Query 参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| partNumber | int | 是 | 当前分片序号（从1开始） |
| uploadId | string | 是 | 上传会话ID（预上传返回） |
| chunk | int | 是 | 当前分片索引（从0开始） |
| chunks | int | 是 | 总分片数 |
| size | int | 是 | 当前分片大小 |
| start | int | 是 | 当前分片起始位置 |
| end | int | 是 | 当前分片结束位置 |
| total | int | 是 | 文件总大小 |

### Headers

| 参数名 | 说明 |
|--------|------|
| Content-Type | `application/octet-stream` |
| Content-Length | 当前分片大小 |
| X-Upos-Auth | 认证令牌（预上传返回的auth） |

### 请求示例

```http
PUT https://upos-sz-mirrorcos.bilivideo.com/ugcfx2lf/n22010101xxx?partNumber=1&uploadId=xxx&chunk=0&chunks=25&size=4194304&start=0&end=4194304&total=104857600
Content-Type: application/octet-stream
Content-Length: 4194304
X-Upos-Auth: xxx

[二进制数据]
```

### 响应示例

```json
{
  "etag": "xxx",
  "partNumber": 1
}
```

---

## 5. 分片确认接口

所有分片上传完成后，确认合并分片。

| 项目 | 内容 |
|------|------|
| **URL** | `https://{endpoint}/{upos_uri}` |
| **方法** | POST |
| **Headers** | `Content-Type: application/json` |
| **Query参数** | 见下表 |
| **Body** | JSON格式 |
| **用途** | 确认所有分片上传完成 |

### Query 参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| output | string | 是 | 固定值 `json` |
| profile | string | 是 | 固定值 `ugcfx/bup` |
| name | string | 是 | 视频文件名 |
| uploadId | string | 是 | 上传会话ID |
| biz_id | string | 是 | 业务ID |

### Headers

| 参数名 | 说明 |
|--------|------|
| Content-Type | `application/json` |
| Origin | `https://member.bilibili.com` |
| Referer | `https://member.bilibili.com/` |

### 请求体

```json
{
  "parts": [
    {
      "partNumber": 1,
      "eTag": "xxx"
    },
    {
      "partNumber": 2,
      "eTag": "xxx"
    }
  ]
}
```

### 请求示例

```http
POST https://upos-sz-mirrorcos.bilivideo.com/ugcfx2lf/n22010101xxx?output=json&profile=ugcfx/bup&name=video.mp4&uploadId=xxx&biz_id=1234567890
Content-Type: application/json
Origin: https://member.bilibili.com
Referer: https://member.bilibili.com/

{
  "parts": [
    {"partNumber": 1, "eTag": "xxx"},
    {"partNumber": 2, "eTag": "xxx"}
  ]
}
```

### 响应示例

```json
{
  "OK": 1,
  "bucket": "ugc",
  "key": "/ugcfx2lf/n22010101xxx",
  "location": "ugcfx2lf/n22010101xxx"
}
```

---

## 6. 视频发布/提交接口

提交视频信息，完成视频发布。这是最后一步。

| 项目 | 内容 |
|------|------|
| **URL** | `https://member.bilibili.com/x/vu/web/add/v3` |
| **方法** | POST |
| **Query参数** | `csrf` |
| **Content-Type** | `application/json` |
| **用途** | 提交视频信息，完成发布 |

### Query 参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| csrf | string | 是 | CSRF Token |

### 请求体参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| copyright | int | 是 | 1:原创 2:转载 |
| cover | string | 是 | 封面URL |
| title | string | 是 | 视频标题 |
| tid | int | 是 | 分区ID |
| tag | string | 是 | 标签，逗号分隔 |
| desc_format_id | int | 否 | 描述格式ID，默认0 |
| desc | string | 否 | 视频简介 |
| source | string | 否 | 视频来源（转载时填写） |
| dynamic | string | 否 | 动态内容 |
| interactive | int | 否 | 互动视频，默认0 |
| videos | array | 是 | 视频文件信息数组 |
| act_reserve_create | int | 否 | 默认0 |
| no_disturbance | int | 否 | 默认0 |
| no_reprint | int | 否 | 1:禁止转载 0:允许 |
| subtitle | object | 否 | 字幕信息 |
| dolby | int | 否 | 杜比音效，默认0 |
| lossless_music | int | 否 | 无损音乐，默认0 |
| csrf | string | 是 | CSRF Token |

### videos 数组元素

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| filename | string | 是 | 视频文件名（分片确认返回的key） |
| title | string | 否 | 分P标题 |
| desc | string | 否 | 分P描述 |

### subtitle 对象

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| open | int | 否 | 是否开启字幕，0或1 |
| lan | string | 否 | 语言代码 |

### 请求示例

```http
POST https://member.bilibili.com/x/vu/web/add/v3?csrf=xxx
Content-Type: application/json
Cookie: SESSDATA=xxx; bili_jct=xxx
Referer: https://member.bilibili.com/video/upload.html

{
  "copyright": 1,
  "cover": "https://i0.hdslb.com/bfs/archive/xxx.jpg",
  "title": "视频标题",
  "tid": 171,
  "tag": "标签1,标签2,标签3",
  "desc_format_id": 0,
  "desc": "这是一个视频简介",
  "source": "",
  "dynamic": "",
  "interactive": 0,
  "videos": [
    {
      "filename": "n22010101xxx",
      "title": "",
      "desc": ""
    }
  ],
  "act_reserve_create": 0,
  "no_disturbance": 0,
  "no_reprint": 1,
  "subtitle": {
    "open": 0,
    "lan": ""
  },
  "dolby": 0,
  "lossless_music": 0,
  "csrf": "xxx"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "aid": 123456789,
    "bvid": "BV1xx411c7mD"
  }
}
```

### 响应字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| aid | int | 视频AV号 |
| bvid | string | 视频BV号 |

---

## 上传流程总结

```
┌─────────────────────────────────────────────────────────────────┐
│                        B站视频上传流程                           │
└─────────────────────────────────────────────────────────────────┘

1. 登录验证
   └── GET https://api.bilibili.com/x/web-interface/nav

2. 上传封面（可选）
   └── POST https://member.bilibili.com/x/vu/web/cover/up

3. 预上传
   └── GET https://member.bilibili.com/preupload
       └── 获取 endpoint, upos_uri, upload_id, auth 等

4. 分片上传（循环）
   └── PUT https://{endpoint}/{upos_uri}
       └── 上传视频分片数据

5. 确认分片
   └── POST https://{endpoint}/{upos_uri}
       └── 发送所有分片的 etag 信息

6. 提交发布
   └── POST https://member.bilibili.com/x/vu/web/add/v3
       └── 完成视频发布，返回 aid 和 bvid
```

---

## 常用分区ID (tid)

| 分区名称 | tid |
|----------|-----|
| 动画-综合 | 27 |
| 动画-短片 | 47 |
| 音乐-原创音乐 | 28 |
| 音乐-翻唱 | 31 |
| 舞蹈-宅舞 | 20 |
| 舞蹈-街舞 | 198 |
| 游戏-单机游戏 | 17 |
| 游戏-电子竞技 | 171 |
| 知识-科学科普 | 201 |
| 知识-人文历史 | 124 |
| 科技-数码 | 95 |
| 科技-软件应用 | 230 |
| 生活-搞笑 | 138 |
| 生活-日常 | 21 |
| 美食-美食制作 | 76 |
| 动物圈-喵星人 | 40 |
| 动物圈-汪星人 | 41 |

---

## 注意事项

1. **Cookie 获取**：需要从浏览器登录 B 站后，复制 `SESSDATA` 和 `bili_jct` 字段
2. **CSRF Token**：即 `bili_jct` 的值
3. **分片大小**：建议按照预上传接口返回的 `chunk_size` 进行分片
4. **并发上传**：可以根据 `threads` 字段设置并发数
5. **重试机制**：分片上传失败时建议进行 3-5 次重试
6. **视频格式**：支持 mp4, flv, avi, mov, wmv 等常见格式
