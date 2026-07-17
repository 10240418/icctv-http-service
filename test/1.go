package main

import "fmt"

func merge(nums1 []int, m int, nums2 []int, n int) {
	// 1,排序
	// 2,合并
	for i := 0; i < m+n; i++ {
		if i == m-1 {
			nums1[i] = nums2[i-m+1]
		}
	}
	nn := len(nums1)
	for i := 0; i < nn; i++ {
		for j := 0; j < nn-i-1; j++ {

			if nums1[j] > nums1[j+1] {
				temp := nums1[j]
				nums1[j] = nums1[j+1]
				nums1[j+1] = temp
			}

		}
	}

}

func main() {
	// Test case 1
	nums1 := []int{1, 2, 3, 0, 0, 0}
	nums2 := []int{2, 5, 6}
	merge(nums1, 3, nums2, 3)
	// Expected: [1, 2, 2, 3, 5, 6]
	fmt.Println(nums1)
}
