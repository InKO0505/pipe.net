package net.pipe.mobile.ui

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.viewmodel.compose.viewModel
import net.pipe.mobile.data.MobileApiRepository
import net.pipe.mobile.ui.screens.ChatShell
import net.pipe.mobile.ui.theme.PipeColors

private val PipeColorScheme = darkColorScheme(
    primary        = PipeColors.Accent,
    onPrimary      = PipeColors.AccentInk,
    background     = PipeColors.Bg,
    onBackground   = PipeColors.Text,
    surface        = PipeColors.Surface1,
    onSurface      = PipeColors.Text,
    surfaceVariant = PipeColors.Surface2,
    outline        = PipeColors.Border,
    error          = PipeColors.Danger,
)

@Composable
fun PipeApp() {
    val context = LocalContext.current
    val repository = remember { MobileApiRepository() }
    val prefs = remember { AppPrefs(context) }
    val viewModel: PipeViewModel = viewModel(factory = PipeViewModel.factory(repository, prefs))

    MaterialTheme(colorScheme = PipeColorScheme) {
        ChatShell(viewModel)
    }
}
