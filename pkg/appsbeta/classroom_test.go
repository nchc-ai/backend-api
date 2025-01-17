package beta

import (
	"context"
	"fmt"
	"testing"

	uuid2 "github.com/google/uuid"
	"github.com/nchc-ai/backend-api/pkg/consts"
	"github.com/nchc-ai/backend-api/pkg/model/db"
	"github.com/nchc-ai/course-crd/pkg/apis/coursecontroller/v1alpha1"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var cm Classroom
var newNS string

func TestMain(m *testing.M) {

	newNS = uuid2.New().String()

	config, err := clientcmd.BuildConfigFromFlags("", "../../conf/minikube-kubeconfig")

	if err != nil {
		fmt.Println("kubeconfig not found, exit...")
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Print("initialize client fail, exit...")
		return
	}

	crdclient, err := versioned.NewForConfig(config)
	if err != nil {
		fmt.Print("initialize crd client fail, exit...")
		return
	}

	// initialize classroom
	cm = Classroom{
		KClientSet:      clientset,
		CourseCrdClient: crdclient,
	}

	m.Run()

	clientset.CoreV1().Namespaces().Delete(context.Background(), consts.AiTrainSystemNamespace, metav1.DeleteOptions{})
	clientset.CoreV1().Namespaces().Delete(context.Background(), newNS, metav1.DeleteOptions{})
}

func TestClassroom_createSecret(t *testing.T) {

	// Test1: both aitrain-system ns & aitrain-system/secret are not exist
	err := cm.copySecretFromSystem(newNS)
	// Should have not found error
	assert.EqualError(t, err, "secrets \"nchc-tls-secret\" not found")

	// Test2: only aitrain-system/secret is not exist
	_, err = cm.KClientSet.CoreV1().Namespaces().Create(
		context.Background(),
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: consts.AiTrainSystemNamespace,
			},
		},
		metav1.CreateOptions{},
	)
	assert.NoError(t, err)
	err = cm.copySecretFromSystem(newNS)
	// Should have not found error
	assert.EqualError(t, err, "secrets \"nchc-tls-secret\" not found")

	// Test3: create aitrain-system/secret, but new ns not exist
	_, err = cm.KClientSet.CoreV1().Secrets(consts.AiTrainSystemNamespace).Create(
		context.Background(),
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: consts.AiTrainSystemNamespace,
				Name:      consts.TlsSecretName,
			},
			Data: map[string][]byte{
				"data": []byte{'a', 'b', 'c'},
			},
			Type: v1.SecretTypeOpaque,
		},
		metav1.CreateOptions{},
	)

	assert.NoError(t, err)
	err = cm.copySecretFromSystem(newNS)
	// should return new ns not found
	assert.EqualError(t, err, fmt.Sprintf("namespaces \"%s\" not found", newNS))

	// Test4: new ns namespace exist
	_, err = cm.KClientSet.CoreV1().Namespaces().Create(
		context.Background(),
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: newNS,
			},
		},
		metav1.CreateOptions{},
	)
	assert.NoError(t, err)
	err = cm.copySecretFromSystem(newNS)
	// should have no error
	assert.NoError(t, err)

	// Test5: compare aitrain-system/secret & <uuid>/secret
	sss, err := cm.KClientSet.CoreV1().Secrets(newNS).Get(context.Background(), consts.TlsSecretName, metav1.GetOptions{})
	assert.NoError(t, err)
	// content must be equal
	assert.Equal(t, "abc", string(sss.Data["data"]))
}

func TestClassroom_updateCourseCRD(t *testing.T) {

	c1, _ := cm.CourseCrdClient.NchcV1alpha1().Courses(newNS).Create(
		context.Background(),
		&v1alpha1.Course{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "c1",
				Namespace: newNS,
			},
			Spec: v1alpha1.CourseSpec{
				Schedule: []string{
					"* * * 1 * *",
				},
			},
		},
		metav1.CreateOptions{},
	)

	c2, _ := cm.CourseCrdClient.NchcV1alpha1().Courses(newNS).Create(
		context.Background(),
		&v1alpha1.Course{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "c2",
				Namespace: newNS,
			},
			Spec: v1alpha1.CourseSpec{
				Schedule: []string{
					"* * * 1 * *",
				},
			},
		},
		metav1.CreateOptions{},
	)

	assert.Equal(t, 1, len(c1.Spec.Schedule))
	assert.Equal(t, 1, len(c2.Spec.Schedule))

	req := db.ClassRoomInfo{
		Model: db.Model{
			ID: newNS,
		},
		ScheduleTime: &db.Schedule{
			CronFormat: []string{
				"* * * 1 * *",
				"* * * 2 * *",
				"* * * 3 * *",
			},
		},
	}

	cm.updateCourseCRD(req)

	c1n, _ := cm.CourseCrdClient.NchcV1alpha1().Courses(newNS).Get(context.Background(), "c1", metav1.GetOptions{})
	c2n, _ := cm.CourseCrdClient.NchcV1alpha1().Courses(newNS).Get(context.Background(), "c2", metav1.GetOptions{})

	assert.Equal(t, 1, len(c1n.Spec.Schedule))
	assert.Equal(t, 1, len(c2n.Spec.Schedule))
}
