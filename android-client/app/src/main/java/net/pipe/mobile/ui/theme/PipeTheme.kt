package net.pipe.mobile.ui.theme

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * pipe.net — design tokens for Compose / Material 3.
 *
 * Single source of truth, keep in sync with handoff/tokens.css and
 * handoff/tokens/design.go.
 *
 * Monochrome palette + ONE accent (acid lime).
 * Use the accent only for: live cursor, active channel rail, send action,
 * owner crown, unread badge, mention pill, online dot.
 */

object PipeColors {
    // Surfaces
    val BgDeep   = Color(0xFF07070A)
    val Bg       = Color(0xFF0A0A0C)
    val Surface1 = Color(0xFF101015)
    val Surface2 = Color(0xFF15151C)
    val Surface3 = Color(0xFF1B1B24)

    // Borders
    val BorderSubtle = Color(0xFF1A1A22)
    val Border       = Color(0xFF22222B)
    val BorderStrong = Color(0xFF2F2F3A)

    // Text
    val Text          = Color(0xFFF1F1EC)
    val TextSecondary = Color(0xFFA3A3A0)
    val TextMuted     = Color(0xFF6F6F6D)
    val TextDim       = Color(0xFF4A4A4D)

    // The accent
    val Accent     = Color(0xFFC8F24A)
    val AccentSoft = Color(0x24C8F24A) // ~14% alpha
    val AccentInk  = Color(0xFF0A0A0C)

    // Roles
    val RoleOwner = Accent
    val RoleAdmin = Text
    val RoleUser  = TextSecondary

    // Status
    val Online = Accent
    val Away   = TextMuted
    val Danger = Color(0xFFF24A4A)
}

object PipeType {
    // Use FontFamily for these in your theme setup.
    const val SansFamily = "Geist"
    const val MonoFamily = "JetBrains Mono"

    // Sizes (mobile)
    val Display    = 22.sp
    val Title      = 16.sp
    val Body       = 14.sp
    val BodyMono   = 13.sp
    val Caption    = 12.sp
    val Meta       = 11.sp
    val Overline   = 10.sp  // letter-spacing 0.18em uppercase

    // Letter spacing scales (Compose uses em via TextUnit; apply in TextStyle)
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

    // Component radii
    val rSmall  = 4.dp   // chips, inputs
    val rMedium = 6.dp   // buttons, list items
    val rLarge  = 10.dp  // panels, cards
    val rXLarge = 14.dp  // sheets

    // Hit targets
    val minTap = 44.dp
}
