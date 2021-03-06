package task

import (
	"bytes"
	"compress/gzip"
	"time"

	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"
	"github.com/mlogclub/simple"
	"github.com/sirupsen/logrus"

	"github.com/mlogclub/bbs-go/common/config"
	"github.com/mlogclub/bbs-go/common/oss"
	"github.com/mlogclub/bbs-go/common/urls"
	"github.com/mlogclub/bbs-go/model"
	"github.com/mlogclub/bbs-go/services"
)

var sitemapBuilding = false

// 生成sitemap
func SitemapTask() {
	if sitemapBuilding {
		logrus.Info("Sitemap in building...")
		return
	}
	sitemapBuilding = true
	defer func() {
		sitemapBuilding = false
	}()

	sm := stm.NewSitemap(1)
	sm.SetDefaultHost(config.Conf.BaseUrl)         // 网站host
	sm.SetSitemapsHost(config.Conf.AliyunOss.Host) // 上传到阿里云所以host设置为阿里云
	sm.SetSitemapsPath("sitemap")                  // sitemap存放目录
	sm.SetVerbose(false)
	sm.SetPretty(false)
	sm.SetCompress(true)
	sm.SetAdapter(&AliyunOssAdapter{})
	sm.Create()

	sm.Add(stm.URL{
		{"loc", "/"},
		{"lastmod", time.Now()},
		{"changefreq", "daily"},
		{"priority", 1.0},
	})

	sm.Add(stm.URL{
		{"loc", "/topics"},
		{"lastmod", time.Now()},
		{"changefreq", "daily"},
		{"priority", 1.0},
	})

	sm.Add(stm.URL{
		{"loc", "/articles"},
		{"lastmod", time.Now()},
		{"changefreq", "daily"},
		{"priority", 1.0},
	})

	sm.Add(stm.URL{
		{"loc", "/projects"},
		{"lastmod", time.Now()},
		{"changefreq", "daily"},
		{"priority", 1.0},
	})

	services.UserService.Scan(func(users []model.User) {
		for _, user := range users {
			userUrl := urls.UserUrl(user.Id)
			sm.Add(stm.URL{
				{"loc", userUrl},
				{"lastmod", time.Now()},
			})
		}
	})

	services.ArticleService.Scan(func(articles []model.Article) bool {
		for _, article := range articles {
			if article.Status == model.StatusOk {
				articleUrl := urls.ArticleUrl(article.Id)
				sm.Add(stm.URL{
					{"loc", articleUrl},
					{"lastmod", simple.TimeFromTimestamp(article.UpdateTime)},
				})
			}
		}
		return true
	})

	services.TopicService.Scan(func(topics []model.Topic) bool {
		for _, topic := range topics {
			if topic.Status == model.StatusOk {
				topicUrl := urls.TopicUrl(topic.Id)
				sm.Add(stm.URL{
					{"loc", topicUrl},
					{"lastmod", simple.TimeFromTimestamp(topic.CreateTime)},
				})
			}
		}
		return true
	})

	services.ProjectService.Scan(func(projects []model.Project) bool {
		for _, project := range projects {
			projectUrl := urls.ProjectUrl(project.Id)
			sm.Add(stm.URL{
				{"loc", projectUrl},
				{"lastmod", simple.TimeFromTimestamp(project.CreateTime)},
			})
		}
		return true
	})

	services.TagService.Scan(func(tags []model.Tag) bool {
		for _, tag := range tags {
			tagUrl := urls.TagArticlesUrl(tag.Id)
			sm.Add(stm.URL{
				{"loc", tagUrl},
				{"lastmod", time.Now()},
				{"changefreq", "daily"},
				{"priority", 0.6},
			})
		}
		return true
	})

	sm.Finalize().PingSearchEngines("http://www.google.cn/webmasters/tools/ping?sitemap=%s")
}

// sitemap上传到aliyun
type AliyunOssAdapter struct {
}

// Bytes gets written content.
func (adp *AliyunOssAdapter) Bytes() [][]byte {
	return nil
}

// Write will create sitemap xml file into the file systems.
func (adp *AliyunOssAdapter) Write(loc *stm.Location, data []byte) {
	var out []byte
	if stm.GzipPtn.MatchString(loc.Filename()) { // 如果需要压缩
		var in bytes.Buffer
		w := gzip.NewWriter(&in)
		_, _ = w.Write(data)
		_ = w.Close()
		out = in.Bytes()
	} else { // 如果不需要压缩
		out = data
	}

	sitemapUrl, err := oss.PutObject(loc.PathInPublic(), out)
	if err != nil {
		logrus.Error("Upload sitemap to aliyun oss error:", err)
	} else {
		logrus.Info("Upload sitemap:", sitemapUrl)
	}
}
