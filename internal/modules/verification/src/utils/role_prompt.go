package utils

import (
	"fmt"
	"strings"

	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/model"
)

func RolePrompt(role string, req *model.VerificationRequest) string {
	var sb strings.Builder

	sb.WriteString(roleInstruction(role))
	sb.WriteString("\n\n")
	sb.WriteString("## Data Laporan\n")
	sb.WriteString(fmt.Sprintf("Judul: %s\n", req.ReportTitle))
	sb.WriteString(fmt.Sprintf("Deskripsi: %s\n", req.ReportDescription))
	sb.WriteString(fmt.Sprintf("Kategori yang diklaim: %s\n", req.ReportCategory))
	sb.WriteString(fmt.Sprintf("Severity yang diklaim: %s\n", req.ReportSeverity))
	sb.WriteString(fmt.Sprintf("Alamat: %s\n", req.ReportAddress))
	sb.WriteString(fmt.Sprintf("Sumber data: %s\n", req.ReportSourceType))

	sb.WriteString("\n## Bukti yang Tersedia\n")
	switch req.ReportSourceType {
	case "user_report":
		if req.ReportPhotoURL != "" {
			sb.WriteString(fmt.Sprintf("Foto laporan: %s\n", req.ReportPhotoURL))
			sb.WriteString("Nilai apakah foto ini secara visual konsisten dengan kategori dan severity yang diklaim.\n")
		} else {
			sb.WriteString("PERINGATAN: Tidak ada foto untuk laporan warga ini. Ini tidak wajar dan patut dicurigai.\n")
		}
	case "ai_news":
		if req.ReportArticleContent != "" {
			sb.WriteString(fmt.Sprintf("Isi artikel berita asli:\n%s\n", req.ReportArticleContent))
			sb.WriteString("Nilai apakah ekstraksi kategori/severity di atas BENAR-BENAR sesuai isi artikel ini, bukan halusinasi/dilebih-lebihkan.\n")
		} else {
			sb.WriteString("PERINGATAN: Tidak ada teks artikel asli yang tersedia untuk verifikasi.\n")
		}
	default:
		sb.WriteString("Tidak ada bukti visual/teks tambahan yang tersedia.\n")
	}

	if req.AdvocateArgument != "" {
		sb.WriteString("\n## Argumen Sebelumnya\n")
		sb.WriteString(fmt.Sprintf("Advocate (mendukung validitas): %q — verdict=%t, confidence=%.2f\n",
			req.AdvocateArgument, req.AdvocateVerdict, req.AdvocateConfidence))
	}
	if req.SkepticArgument != "" {
		sb.WriteString(fmt.Sprintf("Skeptic (menentang validitas): %q — verdict=%t, confidence=%.2f\n",
			req.SkepticArgument, req.SkepticVerdict, req.SkepticConfidence))
	}

	sb.WriteString("\n## Instruksi Output\n")
	sb.WriteString(`Jawab HANYA dengan JSON valid, tanpa teks lain di luar JSON, dengan field:
- verdict (boolean): true jika laporan ini valid untuk ditayangkan
- confidence (number, 0.0-1.0): seberapa yakin kamu dengan verdict ini
- category_slug (string): salah satu dari ["jalan-rusak","jembatan","sampah","bangunan-mangkrak","drainase","lainnya"]
- severity (string): salah satu dari ["ringan","sedang","parah"]
- argument (string): alasan singkat (maksimal 2 kalimat), berdasarkan bukti yang diberikan saja

Jangan mengarang informasi yang tidak ada di data/bukti yang diberikan. Jika bukti tidak cukup untuk menilai, nyatakan itu secara eksplisit dalam argument dan turunkan confidence.`)

	return sb.String()
}

func roleInstruction(role string) string {
	switch role {
	case "advocate":
		return `Kamu adalah Agent Advocate dalam verifikasi laporan infrastruktur publik.
Tugasmu: cari alasan konkret KENAPA laporan ini VALID dan layak dipercaya, berdasarkan bukti yang tersedia.
Jangan memaksakan validitas jika bukti jelas tidak mendukung — kejujuran lebih penting daripada selalu membela.`
	case "skeptic":
		return `Kamu adalah Agent Skeptic dalam verifikasi laporan infrastruktur publik.
Tugasmu: cari alasan konkret KENAPA laporan ini PATUT DICURIGAI — ketidakkonsistenan, indikasi manipulasi, atau kejanggalan lokasi/kategori.
Jangan mengarang kecurigaan jika bukti memang tampak wajar — kejujuran lebih penting daripada selalu curiga.`
	case "manager":
		return `Kamu adalah Agent Manager yang memutuskan validitas akhir laporan ini.
Timbang argumen Advocate dan Skeptic di bawah secara objektif berdasarkan kekuatan bukti masing-masing, bukan berdasarkan siapa yang argumennya lebih panjang atau meyakinkan secara retorika.
Kamu boleh tidak sependapat dengan keduanya jika bukti menunjukkan hal lain.`
	default:
		return fmt.Sprintf("Kamu adalah %s dalam verifikasi laporan infrastruktur publik.", role)
	}
}