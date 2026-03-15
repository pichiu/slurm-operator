// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	slinkyv1beta1 "github.com/SlinkyProject/slurm-operator/api/v1beta1"
	"github.com/SlinkyProject/slurm-operator/internal/utils/testutils"
)

var _ = Describe("Slurm Controller", func() {
	Context("When creating Controller", func() {
		var name = testutils.GenerateResourceName(5)
		var controller *slinkyv1beta1.Controller
		var slurmKeySecret *corev1.Secret
		var jwtKeySecret *corev1.Secret

		BeforeEach(func() {
			slurmKeyRef := testutils.NewSlurmKeyRef(name)
			jwtKeyRef := testutils.NewJwtKeyRef(name)
			slurmKeySecret = testutils.NewSlurmKeySecret(slurmKeyRef)
			jwtKeySecret = testutils.NewJwtKeySecret(jwtKeyRef)
			controller = testutils.NewController(name, slurmKeyRef, jwtKeyRef, nil)
			Expect(k8sClient.Create(ctx, slurmKeySecret.DeepCopy())).To(Succeed())
			Expect(k8sClient.Create(ctx, jwtKeySecret.DeepCopy())).To(Succeed())
			Expect(k8sClient.Create(ctx, controller.DeepCopy())).To(Succeed())
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, controller)
			_ = k8sClient.Delete(ctx, slurmKeySecret)
			_ = k8sClient.Delete(ctx, jwtKeySecret)
		})

		It("Should successfully create create a controller", func(ctx SpecContext) {
			By("Creating Controller CR")
			createdController := &slinkyv1beta1.Controller{}
			controllerKey := client.ObjectKeyFromObject(controller)
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, controllerKey, createdController)).To(Succeed())
			}, testutils.Timeout, testutils.Interval).Should(Succeed())

			By("Expecting Controller CR service")
			serviceKey := controller.ServiceKey()
			service := &corev1.Service{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, serviceKey, service)).To(Succeed())
			}, testutils.Timeout, testutils.Interval).Should(Succeed())

			By("Expecting Controller CR statefulset")
			statefulsetKey := controller.Key()
			statefulset := &appsv1.StatefulSet{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, statefulsetKey, statefulset)).To(Succeed())
			}, testutils.Timeout, testutils.Interval).Should(Succeed())
		}, SpecTimeout(testutils.Timeout))
	})
})
