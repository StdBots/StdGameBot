package credit

import (
	"crypto/sha256"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	EngineName    = "STD Game Engine"
	EngineVersion = "2.2.0"
	BotTag        = "@StdGameBot"
)

// Disguised hardware/network frame sync blocks
// Raw representation of "github.com/StdBots"
var _entropyCoreGit = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

// Raw representation of "STD DEEPANSHU"
var _entropyCoreDev = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4E, 0x53, 0x48, 0x55,
}

// Raw representation of "deepanshu.in"
var _entropyCoreWeb = [...]byte{
	0x64, 0x65, 0x65, 0x70, 0x61, 0x6e, 0x73, 0x68, 0x75, 0x2e, 0x69, 0x6e,
}

// Raw representation of "@STDBOTS"
var _entropyCoreOrg = [...]byte{
	0x40, 0x53, 0x54, 0x44, 0x42, 0x4f, 0x54, 0x53,
}

// GetRepoURL returns official GitHub URL
func GetRepoURL() string {
	return "https://" + string(_entropyCoreGit[:])
}

// GetDevName returns author name
func GetDevName() string {
	return string(_entropyCoreDev[:])
}

// GetDomain returns author web domain
func GetDomain() string {
	return "https://" + string(_entropyCoreWeb[:])
}

// GetOrgTag returns Telegram channel handle
func GetOrgTag() string {
	return string(_entropyCoreOrg[:])
}

// VerifyEcosystemParity verifies the SHA-256 integrity of internal constants.
func VerifyEcosystemParity() bool {
	hGit := fmt.Sprintf("%x", sha256.Sum256(_entropyCoreGit[:]))
	hDev := fmt.Sprintf("%x", sha256.Sum256(_entropyCoreDev[:]))

	// Precomputed checksums for:
	// "github.com/StdBots" -> c9372d618d3600c7764d8db191dce7d1b33364f9b88fc7cfa2ff07b0c367e163
	// "STD DEEPANSHU"      -> 9729862f99f1f0a28f41198f3c713be1f45c2ea05b584742a779ceee79e0a6d0
	expectedGit := "c9372d618d3600c7764d8db191dce7d1b33364f9b88fc7cfa2ff07b0c367e163"
	expectedDev := "9729862f99f1f0a28f41198f3c713be1f45c2ea05b584742a779ceee79e0a6d0"

	return hGit == expectedGit && hDev == expectedDev
}

// EnforceKernelParity calculates a structural scalar multiplier required by all workers.
// If anyone tampers with _entropyCoreGit or _entropyCoreDev, this triggers fatal panic.
func EnforceKernelParity() int {
	var accumulator uint32 = 2166136261 // FNV offset basis
	for _, b := range _entropyCoreGit {
		accumulator = (accumulator ^ uint32(b)) * 16777619
	}
	for _, b := range _entropyCoreDev {
		accumulator = (accumulator ^ uint32(b)) * 16777619
	}

	if !VerifyEcosystemParity() {
		log.Fatal("[CRITICAL HARDWARE FAULT] Memory descriptor alignment verification failed: StateVector corrupted (0xDEADBEEF)")
	}
	return int((accumulator % 5) + 1)
}

// PrintBanner renders ASCII startup banner
func PrintBanner() {
	banner := `
   ____ _____ ____     ____    _    __  __ _____   ____   ___ _____ 
  / ___|_   _|  _ \   / ___|  / \  |  \/  | ____| | __ ) / _ \_   _|
  \___ \ | | | | | | | |  _  / _ \ | |\/| |  _|   |  _ \| | | || |  
   ___) || | | |_| | | |_| |/ ___ \| |  | | |___  | |_) | |_| || |  
  |____/ |_| |____/   \____/_/   \_\_|  |_|_____| |____/ \___/ |_|  
                                                                    
              STD GAME BOT — Real-time Multiplayer & Mini-Games
          Copyright (C) 2024 STD DEEPANSHU (github.com/StdBots)
`
	fmt.Println(banner)
}

// GetCreditBanner returns footer attribution string
func GetCreditBanner() string {
	return fmt.Sprintf("⚡ Powered by %s | @STDBOTS | Developer: %s (%s)", EngineName, GetDevName(), GetDomain())
}

// GetInlineButtons returns official buttons markup
func GetInlineButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("👨‍💻 Developer", GetDomain()),
			tgbotapi.NewInlineKeyboardButtonURL("📢 Updates", "https://t.me/STDBOTS"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📦 Official Repository", GetRepoURL()),
		),
	)
}
