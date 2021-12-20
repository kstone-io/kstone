#  腾讯COS备份参考

## 1 准备工作

+ 前提条件

  - 安装好kstone的Kubernetes集群.
  - 开启COS功能的腾讯云账号. 

  

# 2 参考

### 步骤1: 打开 kstone-dashboard UI 然后点击 "关联集群":

![image-20211220173803261](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220173803261.png)

### 步骤 2: 关联etcd集群到kstone:

本文档将会使用kubeadm管理的etcd集群

填好下面的内容然后 "提交"

![image-20211220174133300](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220174133300.png)

![image-20211220180202214](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220180202214.png)

集群名称: etcd集群的名称，唯一键

集群备注: 集群的备注名称

用于Kubernetes: 该集群是否用于Kubernetes

访问方式:  etcd的访问方式, 如果是 HTTPS, 需要提供相应的证书还有秘钥

CPU核数:  单个节点的CPU, 单位: Core

内存大小:单个节点的内存, 单位: GiB

磁盘类型: 磁盘种类, 比如CLOUD_SSD/CLOUD_PREMIUM/CLOUD_BASIC

磁盘大小: 单个节点的磁盘大小, 单位: GB

集群规模: etcd集群成员数量: 支持1, 3, 5, 7

集群节点映射: etcd集群的内网还有外网地址, 如果不用刻意省略

CA证书: etcd集群的CA证书

客户端证书:  etcd集群的客户端证书

客户端私钥: etcd集群的客户端私钥

描述: etcd的描述

### 步骤 3: 开启备份功能

![image-20211220180651354](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220180651354.png)

现在我们有了一个状态正常的etcd集群, 让我们来开启备份功能.

+ 我们首先需要拿到腾讯云COS的SecretId还有SecretKey还有bucket
  - 打开并登录腾讯云COS对象存储网站: https://console.cloud.tencent.com/cos5
  - 如果没有开启该功能，请开启
  - 创建一个bucket然后保存下该bucket的URL路径
  - 打开并登录腾讯云CAM对象存储网站: https://console.cloud.tencent.com/cam
  - 创建一个具有QcloudCOSDataFullControl权限的用户并且保存下该用户的SecretId还有SecretKey

+ 让我们回到 kstone-dashboard

  - 点击 "操作"

  - 点击 "集群功能项"

  - 点击 "Backup:"

  - 填好下面的参数

  - BackupIntervalInSecond 决定了多久backup-operator会进行一次etcd备份工作

  - MaxBackups 是你想要存放的etcd备份文件的最大数量

  - TimeoutInSecond 是 backup-operator的超时时间

  - SecretId 是之前步骤保存下来的SecretId 

  - SecretKey 是之前步骤保存下来的SecretKey 

  - Path:

    # 重要!!! 

    你首先需要去掉 https:// 然后加上  / 还有你想要的文件的备份前缀

    如果你的COS bucket URL 地址是 https://etcd-bakcup-XXXXXXX.cos.ap-guangzhou.myqcloud.com 

    然后你想要你的备份文件的前缀是my-first-cluster, 你需要像下面一样填好path:

    #### etcd-bakcup-XXXXXXX.cos.ap-guangzhou.myqcloud.com/my-first-cluster
    
    
  
  ![image-20211220182218324](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220182218324.png)



![image-20211220182246325](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220182246325.png)

![image-20211220183506017](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220183506017.png)



+ 等待BackupIntervalInSecond秒然后检查 COS bucket

  ![image-20211220184047124](C:\Users\maudlin\AppData\Roaming\Typora\typora-user-images\image-20211220184047124.png)

你会发现一个带有之前指定前缀的etcd备份文件