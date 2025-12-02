package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type baseResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func respondData(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(baseResponse{
		Success: true,
		Data:    data,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(baseResponse{
		Success: false,
		Error:   message,
	})
}

func respondResult(w http.ResponseWriter, err error, successStatus int, payload interface{}) {
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, successStatus, payload)
}

func decodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty body")
	}
	return json.Unmarshal(body, dst)
}

func parseIDFromQuery(r *http.Request) (int64, error) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return 0, errors.New("id is required")
	}
	return strconv.ParseInt(idStr, 10, 64)
}

// handleServiceError 根据服务层错误返回合适的HTTP状态码和错误信息
func handleServiceError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	// 导入services包以访问错误常量
	// 注意：这里使用字符串匹配，因为不能直接导入services包（避免循环依赖）
	errMsg := err.Error()

	// 根据错误消息判断错误类型
	switch errMsg {
	case "building not found", "orangepi not found", "nvr not found":
		respondError(w, http.StatusNotFound, errMsg)
	case "already bound to another building", "not bound to any building", "building has no ismart_id":
		respondError(w, http.StatusBadRequest, errMsg)
	default:
		// 其他错误默认为500
		respondError(w, http.StatusInternalServerError, errMsg)
	}
}
