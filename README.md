# Omni-repository
![GitHub go.mod Go version (branch & subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/luonannet/omni-repository/main) ![GitHub](https://img.shields.io/github/license/luonannet/omni-repository)

本项目是为OmniBuildPlatform 社区提供基础存储服务而创建。
一方面存储用户用于构建openEuler系统所需要的基础images文件，另一方面存储用户构建完成后产生的ISO文件。


## 功能列表
- [Omni-repository](#omni-repository)
  - [功能列表](#功能列表)
      - [上传iso文件](#上传iso文件)
      - [异步缓存iso](#异步缓存iso)
      - [文件凭证使用](#文件凭证使用)
      - [文件获取](#文件获取)
  - [Contributing](#contributing)
  - [License](#license)

#### 上传iso文件
`/upload` 使用form表单将iso文件post到/upload 接口。
#### 异步缓存iso
`/loadfrom` 凭token 输入文件url地址后会得到返回凭证ID，系统在后台自动下载你的文件，并和你提供的checksum（sha256）进行验证。
#### 文件凭证使用
`/query` 使用凭证ID 即可在omni-manager 构建系统中使用你之前下载的iso文件。或用于构建新系统，或用于安装。

#### 文件获取
 `/browse` 在被授权获得文件的确切url地址后，可以使用本接口获取你的文件。

## Contributing


 

## License

[MIT © OmniBuildPlatfrom](../LICENSE)