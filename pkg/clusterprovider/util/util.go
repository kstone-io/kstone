package util

import "strconv"

const (
	QosBurstable  = "Burstable"
	QosGuaranteed = "Guaranteed"
)

func CalculateRequestCPU(cpu, ratio uint) string {
	request := 1000 * cpu / ratio
	requestCPU := strconv.Itoa(int(request)) + "m"
	return requestCPU
}

func CalculateRequestMem(mem, ratio uint) string {
	request := 1024 * mem / ratio
	requestMem := strconv.Itoa(int(request)) + "Mi"
	return requestMem
}

func ConvertRequestCPU(cpu string) (string, error) {
	tmpRequestCPU, err := strconv.Atoi(cpu)
	if err != nil {
		return "", err
	}
	tmpRequestCPU *= 1000
	return strconv.Itoa(tmpRequestCPU) + "m", nil
}

func ConvertRequestMem(mem string) (string, error) {
	tmpMem, err := strconv.Atoi(mem)
	if err != nil {
		return "", err
	}
	tmpMem *= 1024
	return strconv.Itoa(tmpMem) + "Mi", nil

}
