package a2a

import "fmt"

func ParsePaymentProofHeader(header string) (PaymentProof, error) {
	if header == "" {
		return PaymentProof{}, fmt.Errorf("Payment-Proof header is required")
	}
	var proof PaymentProof
	if err := DecodeBase64JSON(header, &proof); err != nil {
		return PaymentProof{}, fmt.Errorf("decode Payment-Proof: %w", err)
	}
	if proof.Protocol.PaymentProof == "" {
		return PaymentProof{}, fmt.Errorf("payment_proof is required")
	}
	if proof.Protocol.TradeNo == "" {
		return PaymentProof{}, fmt.Errorf("trade_no is required")
	}
	if proof.Method.ClientSession == "" {
		return PaymentProof{}, fmt.Errorf("client_session is required")
	}
	return proof, nil
}
