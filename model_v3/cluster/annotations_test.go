package cluster

import (
	"reflect"
	"testing"
)

const lastAppliedKey = "kubectl.kubernetes.io/last-applied-configuration"

func TestReportableAnnotations(t *testing.T) {
	tests := []struct {
		name string
		src  map[string]string
		want map[string]string
	}{
		{name: "nil 原样返回", src: nil, want: nil},
		{
			name: "不含清单副本时原样返回",
			src:  map[string]string{"deployment.kubernetes.io/revision": "3"},
			want: map[string]string{"deployment.kubernetes.io/revision": "3"},
		},
		{
			name: "去掉 kubectl 的清单副本，其余保留",
			src: map[string]string{
				lastAppliedKey:                      `{"kind":"Deployment"}`,
				"deployment.kubernetes.io/revision": "3",
			},
			want: map[string]string{"deployment.kubernetes.io/revision": "3"},
		},
		{
			name: "其他工具写入的清单副本同样去掉",
			src: map[string]string{
				"kapp.k14s.io/original":           "{}",
				"objectset.rio.cattle.io/applied": "H4sI",
				"banzaicloud.com/last-applied":    "UEsD",
				"team":                            "platform",
			},
			want: map[string]string{"team": "platform"},
		},
		{
			name: "发行版自带部署器写的同名后缀注解也去掉",
			src: map[string]string{
				"k0s.k0sproject.io/last-applied-configuration": `{"kind":"Deployment"}`,
				"example.io/last-applied":                      "H4sI",
				"team":                                         "platform",
			},
			want: map[string]string{"team": "platform"},
		},
		{
			name: "键名没见过、但值是一份序列化的 K8s 对象，同样去掉",
			src: map[string]string{
				"some-tool.example.io/snapshot": ` {"apiVersion":"apps/v1","kind":"Deployment","spec":{}}`,
				"team":                          "platform",
			},
			want: map[string]string{"team": "platform"},
		},
		{
			name: "值是普通 JSON（不是 K8s 对象）时保留",
			src: map[string]string{
				"example.io/config": `{"timeout":"30s","kind":"fast"}`,
				"example.io/notes":  "apiVersion and kind are mentioned here",
			},
			want: map[string]string{
				"example.io/config": `{"timeout":"30s","kind":"fast"}`,
				"example.io/notes":  "apiVersion and kind are mentioned here",
			},
		},
		{
			name: "只有清单副本时返回 nil",
			src:  map[string]string{lastAppliedKey: "{}"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReportableAnnotations(tt.src)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReportableAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 调用方传入的是 K8s 客户端对象上的 map，不得被就地修改。
func TestReportableAnnotations_DoesNotMutateSource(t *testing.T) {
	src := map[string]string{lastAppliedKey: "{}", "team": "platform"}
	ReportableAnnotations(src)
	if _, ok := src[lastAppliedKey]; !ok || len(src) != 2 {
		t.Errorf("源 map 被修改: %v", src)
	}
}

// 快照里凡是名为 Annotations 的字段都必须被 DropManifestCopyAnnotations 覆盖。
// 用反射给每个这样的字段填值，新增资源类型或嵌套层级时漏掉清理，这里会失败。
func TestDropManifestCopyAnnotations_CoversEveryAnnotationField(t *testing.T) {
	snapshot := &ClusterSnapshot{}
	filled := fillAnnotationFields(reflect.ValueOf(snapshot).Elem(), 0)
	if filled == 0 {
		t.Fatal("没有填到任何 Annotations 字段，测试装置失效")
	}

	snapshot.DropManifestCopyAnnotations()

	var visited int
	walkAnnotationFields(reflect.ValueOf(snapshot).Elem(), "ClusterSnapshot", func(path string, annotations map[string]string) {
		visited++
		if len(annotations) != 1 {
			t.Errorf("%s 仍带有清单副本注解: %v", path, annotations)
		}
		if annotations["team"] != "platform" {
			t.Errorf("%s 的普通注解丢失: %v", path, annotations)
		}
	})
	if visited != filled {
		t.Errorf("填了 %d 个 Annotations 字段，回查到 %d 个", filled, visited)
	}
}

func TestDropManifestCopyAnnotations_NilSnapshot(t *testing.T) {
	var snapshot *ClusterSnapshot
	snapshot.DropManifestCopyAnnotations() // 不得 panic
}

var annotationMapType = reflect.TypeOf(map[string]string{})

// fillAnnotationFields 递归遍历结构体：每个切片放一个元素，每个指针分配实例，
// 每个 Annotations 字段写入一条清单副本和一条普通注解。返回填充的字段数。
func fillAnnotationFields(v reflect.Value, depth int) int {
	if depth > 8 {
		return 0
	}
	switch v.Kind() {
	case reflect.Struct:
		filled := 0
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue
			}
			if v.Type().Field(i).Name == "Annotations" && field.Type() == annotationMapType {
				field.Set(reflect.ValueOf(map[string]string{
					lastAppliedKey: "{}",
					"k0s.k0sproject.io/last-applied-configuration": "{}",
					"some-tool.example.io/snapshot":                `{"apiVersion":"v1","kind":"Pod"}`,
					"team":                                         "platform",
				}))
				filled++
				continue
			}
			filled += fillAnnotationFields(field, depth+1)
		}
		return filled
	case reflect.Slice:
		elemKind := v.Type().Elem().Kind()
		if elemKind != reflect.Struct && elemKind != reflect.Ptr {
			return 0
		}
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		return fillAnnotationFields(v.Index(0), depth+1)
	case reflect.Ptr:
		if v.Type().Elem().Kind() != reflect.Struct {
			return 0
		}
		v.Set(reflect.New(v.Type().Elem()))
		return fillAnnotationFields(v.Elem(), depth+1)
	}
	return 0
}

func walkAnnotationFields(v reflect.Value, path string, visit func(path string, annotations map[string]string)) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			name := v.Type().Field(i).Name
			field := v.Field(i)
			if name == "Annotations" && field.Type() == annotationMapType {
				visit(path+"."+name, field.Interface().(map[string]string))
				continue
			}
			walkAnnotationFields(field, path+"."+name, visit)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkAnnotationFields(v.Index(i), path+"[]", visit)
		}
	case reflect.Ptr:
		if !v.IsNil() {
			walkAnnotationFields(v.Elem(), path, visit)
		}
	}
}
