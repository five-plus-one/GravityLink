package base62

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Encode(n uint64) string {
	if n == 0 {
		return "0"
	}

	buf := make([]byte, 0, 11)
	for n > 0 {
		remainder := n % 62
		buf = append(buf, alphabet[remainder])
		n /= 62
	}

	for left, right := 0, len(buf)-1; left < right; left, right = left+1, right-1 {
		buf[left], buf[right] = buf[right], buf[left]
	}
	return string(buf)
}
