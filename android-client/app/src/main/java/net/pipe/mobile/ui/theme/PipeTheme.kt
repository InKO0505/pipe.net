package net.pipe.mobile.ui.theme

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * pipe.net — design tokens for Compose / Material 3.
 * Single source of truth — keep in sync with handoff/tokens.css and handoff/tokens/design.go.
 * Monochrome palette + ONE accent (acid lime #c8f24a).
 */
object PipeColors {
    val BgDeep   = Color(0xFF07070A)
    val Bg       = Color(0xFF0A0A0C)
    val Surface1 = Color(0xFF101015)
    val Surface2 = Color(0xFF15151C)
    val Surface3 = Color(0xFF1B1B24)

    val BorderSubtle = Color(0xFF1A1A22)
    val Border       = Color(0xFF22222B)
    val BorderStrong = Color(0xFF2F2F3A)

    val Text          = Color(0xFFF1F1EC)
    val TextSecondary = Color(0xFFA3A3A0)
    val TextMuted     = Color(0xFF6F6F6D)
    val TextDim       = Color(0xFF4A4A4D)

    val Accent     = Color(0xFFC8F24A)
    val AccentSoft = Color(0x24C8F24A)
    val AccentInk  = Color(0xFF0A0A0C)

    val RoleOwner = Accent
    val RoleAdmin = Text
    val RoleUser  = TextSecondary

    val Online = Accent
    val Away   = TextMuted
    val Danger = Color(0xFFF24A4A)
}

object PipeType {
    const val MonoFamily = "JetBrains Mono"

    val Display  = 22.sp
    val Title    = 16.sp
    val Body     = 14.sp
    val BodyMono = 13.sp
    val Caption  = 12.sp
    val Meta     = 11.sp
    val Overline = 10.sp

    const val DisplayTracking  = -0.02f
    const val OverlineTracking = 0.18f
}

object PipeSpace {
    val xs    = 4.dp
    val sm    = 8.dp
    val md    = 12.dp
    val lg    = 16.dp
    val xl    = 22.dp
    val xxl   = 32.dp

    val rSmall  = 4.dp
    val rMedium = 6.dp
    val rLarge  = 10.dp
    val rXLarge = 14.dp

    val minTap = 44.dp
}
