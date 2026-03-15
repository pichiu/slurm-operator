// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package token

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	slinkyv1beta1 "github.com/SlinkyProject/slurm-operator/api/v1beta1"
	"github.com/SlinkyProject/slurm-operator/internal/utils/testutils"
)

var _ = Describe("Token Controller", func() {
	Context("When reconciling a Token", func() {
		var name = testutils.GenerateResourceName(5)
		var token *slinkyv1beta1.Token
		var jwtKeySecret *corev1.Secret

		BeforeEach(func() {
			jwtKeyRef := testutils.NewJwtKeyRef(name)
			jwtKeySecret = testutils.NewJwtKeySecret(jwtKeyRef)
			token = testutils.NewToken(name, jwtKeySecret)
			Expect(k8sClient.Create(ctx, jwtKeySecret.DeepCopy())).To(Succeed())
			Expect(k8sClient.Create(ctx, token.DeepCopy())).To(Succeed())
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, jwtKeySecret)
			_ = k8sClient.Delete(ctx, token)
		})

		It("Should successfully create create a token", func(ctx SpecContext) {
			By("Creating Token CR")
			createdToken := &slinkyv1beta1.Token{}
			tokenKey := client.ObjectKeyFromObject(token)
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, tokenKey, createdToken)).To(Succeed())
			}).Should(Succeed())
		}, SpecTimeout(testutils.Timeout))
	})
})
