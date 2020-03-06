package sm2

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"testing"
	"time"
)

func TestSm2(t *testing.T) {
	priv, err := GenerateKey() // 生成密钥对
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v\n", priv.Curve.IsOnCurve(priv.X, priv.Y)) // 验证是否为sm2的曲线
	pub := &priv.PublicKey
	msg := []byte("123456")
	d0, err := pub.Encrypt(msg)
	if err != nil {
		fmt.Printf("Error: failed to encrypt %s: %v\n", msg, err)
		return
	}
	// fmt.Printf("Cipher text = %v\n", d0)
	d1, err := priv.Decrypt(d0)
	if err != nil {
		fmt.Printf("Error: failed to decrypt: %v\n", err)
	}
	fmt.Printf("clear text = %s\n", d1)
	ok, err := WritePrivateKeytoPem("priv.pem", priv, nil) // 生成密钥文件
	if ok != true {
		log.Fatal(err)
	}
	pubKey, _ := priv.Public().(*PublicKey)
	ok, err = WritePublicKeytoPem("pub.pem", pubKey, nil) // 生成公钥文件
	if ok != true {
		log.Fatal(err)
	}
	msg = []byte("test")
	err = ioutil.WriteFile("ifile", msg, os.FileMode(0644)) // 生成测试文件
	if err != nil {
		log.Fatal(err)
	}
	privKey, err := ReadPrivateKeyFromPem("priv.pem", nil) // 读取密钥
	if err != nil {
		log.Fatal(err)
	}
	pubKey, err = ReadPublicKeyFromPem("pub.pem", nil) // 读取公钥
	if err != nil {
		log.Fatal(err)
	}
	msg, _ = ioutil.ReadFile("ifile")                // 从文件读取数据
	sign, err := privKey.Sign(rand.Reader, msg, nil) // 签名
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("ofile", sign, os.FileMode(0644))
	if err != nil {
		log.Fatal(err)
	}
	signdata, _ := ioutil.ReadFile("ofile")
	ok = privKey.Verify(msg, signdata) // 密钥验证
	if ok != true {
		fmt.Printf("Verify error\n")
	} else {
		fmt.Printf("Verify ok\n")
	}
	ok = pubKey.Verify(msg, signdata) // 公钥验证
	if ok != true {
		fmt.Printf("Verify error\n")
	} else {
		fmt.Printf("Verify ok\n")
	}
	templateReq := CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "test.example.com",
			Organization: []string{"Test"},
		},
		//		SignatureAlgorithm: ECDSAWithSHA256,
		SignatureAlgorithm: SM2WithSM3,
	}
	_, err = CreateCertificateRequestToPem("req.pem", &templateReq, privKey)
	if err != nil {
		log.Fatal(err)
	}
	req, err := ReadCertificateRequestFromPem("req.pem")
	if err != nil {
		log.Fatal(err)
	}
	err = req.CheckSignature()
	if err != nil {
		log.Fatalf("Request CheckSignature error:%v", err)
	} else {
		fmt.Printf("CheckSignature ok\n")
	}
	testExtKeyUsage := []ExtKeyUsage{ExtKeyUsageClientAuth, ExtKeyUsageServerAuth}
	testUnknownExtKeyUsage := []asn1.ObjectIdentifier{[]int{1, 2, 3}, []int{2, 59, 1}}
	extraExtensionData := []byte("extra extension")
	commonName := "test.example.com"
	template := Certificate{
		// SerialNumber is negative to ensure that negative
		// values are parsed. This is due to the prevalence of
		// buggy code that produces certificates with negative
		// serial numbers.
		SerialNumber: big.NewInt(-1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"TEST"},
			Country:      []string{"China"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  []int{2, 5, 4, 42},
					Value: "Gopher",
				},
				// This should override the Country, above.
				{
					Type:  []int{2, 5, 4, 6},
					Value: "NL",
				},
			},
		},
		NotBefore: time.Unix(1000, 0),
		NotAfter:  time.Unix(100000, 0),

		//		SignatureAlgorithm: ECDSAWithSHA256,
		SignatureAlgorithm: SM2WithSM3,

		SubjectKeyID: []byte{1, 2, 3, 4},
		KeyUsage:     KeyUsageCertSign,

		ExtKeyUsage:        testExtKeyUsage,
		UnknownExtKeyUsage: testUnknownExtKeyUsage,

		BasicConstraintsValid: true,
		IsCA:                  true,

		OCSPServer:            []string{"http://ocsp.example.com"},
		IssuingCertificateURL: []string{"http://crt.example.com/ca1.crt"},

		DNSNames:       []string{"test.example.com"},
		EmailAddresses: []string{"gopher@golang.org"},
		IPAddresses:    []net.IP{net.IPv4(127, 0, 0, 1).To4(), net.ParseIP("2001:4860:0:2001::68")},

		PolicyIdentifiers:   []asn1.ObjectIdentifier{[]int{1, 2, 3}},
		PermittedDNSDomains: []string{".example.com", "example.com"},

		CRLDistributionPoints: []string{"http://crl1.example.com/ca1.crl", "http://crl2.example.com/ca1.crl"},

		ExtraExtensions: []pkix.Extension{
			{
				Id:    []int{1, 2, 3, 4},
				Value: extraExtensionData,
			},
			// This extension should override the SubjectKeyId, above.
			{
				Id:       oidExtensionSubjectKeyID,
				Critical: false,
				Value:    []byte{0x04, 0x04, 4, 3, 2, 1},
			},
		},
	}
	pubKey, _ = priv.Public().(*PublicKey)
	ok, _ = CreateCertificateToPem("cert.pem", &template, &template, pubKey, privKey)
	if ok != true {
		fmt.Printf("failed to create cert file\n")
	}
	cert, err := ReadCertificateFromPem("cert.pem")
	if err != nil {
		fmt.Printf("failed to read cert file")
	}
	err = cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("CheckSignature ok\n")
	}
}

//对比两种[]byte转公钥
func Test_Com2BytesToPublicKey(t *testing.T) {
	//公钥
	/*pub, _ := hex.DecodeString("F6D326509BA8DA09AA34CD85AEF79DBA45FD17E675541B15EF5EC9B8F4AB18BCA13A2F04C6BC1607CA72CC296A9ACF7BF26891C32B210B947CA88F3B92801E8F")
	pubKey := Decompress([]byte(pub))
	t.Log(pubKey)
	//t.Log(err)
	pubKey1, _ := RawBytesToPublicKey(pub)
	t.Log(pubKey1)*/
	pubKey, err := ReadPublicKeyFromPem("pub.pem", nil) // 读取公钥
	if err != nil {
		log.Fatal(err)
	}
	pubKey1, err1 := MarshalSm2PublicKey(pubKey)
	t.Log(pubKey1, err1)

	pubKey3 := Compress(pubKey)
	t.Log(pubKey3)
	/*pubKey2,err2 := ParseSm2PublicKey(pubKey1)
	t.Log(pubKey2,err2)*/
}

//测试sm2数据（根据国家密码管理局提供数据验证是否为国密算法）
func TestRawBytesToPrivateKey(t *testing.T) {
	//公钥
	pu, _ := hex.DecodeString("F6D326509BA8DA09AA34CD85AEF79DBA45FD17E675541B15EF5EC9B8F4AB18BCA13A2F04C6BC1607CA72CC296A9ACF7BF26891C32B210B947CA88F3B92801E8F")
	pubKey, _ := RawBytesToPublicKey(pu)
	t.Log(pubKey)
	//私钥
	/*pr, _ := hex.DecodeString("9ED45E25826B67E8ACCD63D3E1605179CF417E2ADB391361CDBF367EC1687ECE")
	priKey, _ := RawBytesToPrivateKey(pr)
	key1 := priKey.Public()
	t.Log(priKey)
	t.Log(key1)
	t.Log(priKey.Y)
	t.Log(priKey.X)
	t.Log(priKey.D)
	msg := []byte("67")
	//加密
	s, _ := Encrypt(pubKey, msg)
	t.Log(s)
	//解密
	d, _ := Decrypt(priKey, s)
	t.Log(msg)
	t.Log(d)*/
}

//5000	    284952 ns/op
func BenchmarkGenerateKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateKey()
	}
}

//BenchmarkSM2-8   	    1000	   2115599 ns/op	  128243 B/op	    1845 allocs/op
func BenchmarkSM2(t *testing.B) {
	t.ReportAllocs()
	msg := []byte("test")
	priv, err := GenerateKey() // 生成密钥对
	if err != nil {
		log.Fatal(err)
	}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		sign, err := priv.Sign(rand.Reader, msg, nil) // 签名
		if err != nil {
			log.Fatal(err)
		}
		priv.Verify(msg, sign) // 密钥验证
		// if ok != true {
		// 	fmt.Printf("Verify error\n")
		// } else {
		// 	fmt.Printf("Verify ok\n")
		// }
	}
}

var prk, _ = GenerateKey()
var testMessage = "abc7454480385556955618697817098332954057395722449793480218377692479888opq"

//20000000	        126 ns/op
func BenchmarkP256Sm2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		P256Sm2()
	}
}

//2000000000	         0.39 ns/op
func BenchmarkPublic(b *testing.B) {

	for i := 0; i < b.N; i++ {
		prk.Public()
	}
}

//0.5 ns/op
func TestPublic(t *testing.T) {
	start := time.Now()
	flag := false
	for i := 1; i <= 2000000000; i++ {
		prk.Public()
		if i == 2000000000 {
			flag = true
		}
	}
	if flag == true {
		fmt.Printf("结束%v \n", time.Since(start))
		fmt.Println(flag)
	}
}

//5000	    326674 ns/op	    6143 B/op	     107 allocs/op
func BenchmarkSign(t *testing.B) {
	t.ReportAllocs()
	msg := []byte(testMessage)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		prk.Sign(rand.Reader, msg, nil) // 签名
	}
}

func TestCompress(t *testing.T) {
	priv, _ := GenerateKey() // 生成密钥对
	rel := Compress(&priv.PublicKey)
	t.Log(rel)
}

//5000000	       286 ns/op
func BenchmarkCompress(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Compress(&prk.PublicKey)
	}
}

//1000	   1312490 ns/op
func BenchmarkDecrypt(b *testing.B) {
	pri, _ := GenerateKey()
	for i := 0; i < b.N; i++ {
		Decrypt(pri, []byte("12341324564323215512311111111545123413245643232155123111111115456"))
	}
}

//1000	   1595242 ns/op
func BenchmarkEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encrypt(&prk.PublicKey, []byte("123456"))
	}
}
