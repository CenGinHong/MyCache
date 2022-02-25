package geecache

import (
	"MyCache/geecache/consistent"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const defaultBasePath = "/_geecache/"
const defaultReplicas = 50

// HTTPPool 处理外界请求
type HTTPPool struct {
	self       string // 记录自己的地址，包括主机/ip和端口
	basePath   string // 节点间通讯地址的前缀，http://example.com/_geecache/ 就用于节点间的访问
	mu         sync.Mutex
	peers      *consistent.Map
	httpGetter map[string]*httpGetter // key eg:http://10.0.0.2:8008，向其他缓存节点中获取内容
}

// PickPeer 在哈希环上选出key对应的节点
func (p *HTTPPool) PickPeer(key string) (peer PeerGetter, ok bool) {
	// 上锁
	p.mu.Lock()
	defer p.mu.Unlock()
	// 根据key值从哈希环获取落在那个节点上
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetter[peer], true
	}
	return nil, false
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打日志
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServerHTTP 处理http请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 必须要有该前缀
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	// 打印地址
	p.Log("%s %s", r.Method, r.URL.Path)
	// 应该呈现下面这种地址结构获取缓存信息
	// /<basepath>/<groupname>/<key> required
	// 切分
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// 获取缓存名
	groupName := parts[0]
	// 缓存键名
	key := parts[1]
	// 过去缓存
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	// 获取key值
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// 写回请求
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err = w.Write(view.ByteSlice()); err != nil {
		return
	}
}

// httpGetter 从其他节点中获取缓存内容
type httpGetter struct {
	// 即将访问的远程节点的地址，eg:http://example.com/_geecache
	baseURL string
}

// Get 获取返回值，转换为bytes类型
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 组成url
	u := fmt.Sprintf(
		"%v/%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key))
	// 调用Get 方法请求向其他节点获取节点信息
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			log.Print(err)
		}
	}(res.Body)
	// 错误处理
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	// 读取返回体结构
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}
	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)
