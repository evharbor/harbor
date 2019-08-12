Harbor webserver developed by golang   
Gin + Gorm

#### 依赖包被墙的问题
(1) 包github.com/swaggo/swag/cmd/swag依赖包golang.org/x/text。  
>下载github.com/golang/text放到路径{gopath}/src/golang.org/x/下。    

(2) github.com/swaggo/gin-swagger依赖golang.org/x/net。   
> 下载github.com/golang/net放到路径{gopath}/src/golang.org/x/下。
