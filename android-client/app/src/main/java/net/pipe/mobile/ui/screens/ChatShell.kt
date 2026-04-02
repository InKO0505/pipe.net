package net.pipe.mobile.ui.screens

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.pipe.mobile.data.ChatSummary
import net.pipe.mobile.data.MessageItem
import net.pipe.mobile.data.ModLogItem
import net.pipe.mobile.data.UserSummary
import net.pipe.mobile.ui.PipeUiState
import net.pipe.mobile.ui.PipeViewModel
import net.pipe.mobile.ui.SheetContent

private object PipePalette {
    val Background = Color(0xFF070B1C)
    val BackgroundElevated = Color(0xFF0C1230)
    val Panel = Color(0xFF101734)
    val PanelRaised = Color(0xFF141E42)
    val PanelSoft = Color(0xFF182552)
    val Outline = Color(0xFF27335F)
    val OutlineBright = Color(0xFF395095)
    val TextPrimary = Color(0xFFF5F7FF)
    val TextSecondary = Color(0xFFAFB8DD)
    val TextMuted = Color(0xFF8090BD)
    val AccentCyan = Color(0xFF1FD1FF)
    val AccentBlue = Color(0xFF2F84FF)
    val AccentViolet = Color(0xFF9242FF)
    val AccentLilac = Color(0xFFC19BFF)
    val Success = Color(0xFF2ED3A5)
    val Warning = Color(0xFFFFB454)
    val Danger = Color(0xFFFF6D8D)
    val MineBubble = Color(0xFF14285A)
    val OtherBubble = Color(0xFF0D1530)
    val Selection = Color(0xFF1A2654)
    val SoftCyan = Color(0x3323CBFF)
    val SoftViolet = Color(0x339242FF)
    val SoftDanger = Color(0x33FF6D8D)
}

private val PipePanelShape = RoundedCornerShape(28.dp)
private val PipeCardShape = RoundedCornerShape(24.dp)

@Composable
fun ChatShell(viewModel: PipeViewModel) {
    val state by viewModel.uiState.collectAsState()

    Surface(
        modifier = Modifier.fillMaxSize(),
        color = PipePalette.Background,
    ) {
        if (state.token == null) {
            ConnectScreen(
                state = state,
                onEndpointChange = viewModel::updateEndpoint,
                onUsernameChange = viewModel::updateUsername,
                onConnect = viewModel::connect,
            )
        } else {
            ChatScreen(
                state = state,
                onDraftChange = viewModel::updateComposerText,
                onRefresh = viewModel::refresh,
                onShowMentions = viewModel::showMentions,
                onOpenMembers = viewModel::openMembers,
                onOpenModLog = viewModel::openModLog,
                onDmInputChange = viewModel::updateQuickDmInput,
                onCreateDm = viewModel::createDirectMessage,
                onLookupUser = viewModel::lookupUser,
                onCloseSheet = viewModel::closeSheet,
                onChatClick = viewModel::selectChat,
                onSelectMessage = viewModel::selectMessage,
                onStartReply = viewModel::startReplyToSelectedMessage,
                onStartEdit = viewModel::startEditingSelectedMessage,
                onCancelEdit = viewModel::cancelEditing,
                onDeleteSelected = viewModel::deleteSelectedMessage,
                onSend = viewModel::submitComposer,
                onNewChannelNameChange = viewModel::updateNewChannelName,
                onToggleNewChannelPrivacy = viewModel::toggleNewChannelPrivacy,
                onCreateChannel = viewModel::createChannel,
                onMemberInputChange = viewModel::updateMemberActionInput,
                onInviteMember = viewModel::inviteMember,
                onRemoveMember = viewModel::removeMember,
            )
        }
    }
}

@Composable
private fun ConnectScreen(
    state: PipeUiState,
    onEndpointChange: (String) -> Unit,
    onUsernameChange: (String) -> Unit,
    onConnect: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(screenBackgroundBrush())
            .statusBarsPadding()
            .navigationBarsPadding()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 20.dp, vertical = 18.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        Spacer(modifier = Modifier.height(18.dp))
        PipeBrandMark(modifier = Modifier.align(Alignment.CenterHorizontally))
        Spacer(modifier = Modifier.height(20.dp))
        Text(
            text = "Pipe Net",
            modifier = Modifier.align(Alignment.CenterHorizontally),
            color = PipePalette.TextPrimary,
            style = MaterialTheme.typography.headlineLarge,
            fontWeight = FontWeight.Black,
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "A sharper mobile shell for your local network chat server.",
            modifier = Modifier.align(Alignment.CenterHorizontally),
            color = PipePalette.TextSecondary,
            style = MaterialTheme.typography.bodyLarge,
        )
        Spacer(modifier = Modifier.height(22.dp))
        PipePanel(modifier = Modifier.fillMaxWidth(), accent = true) {
            Text(
                text = "Connect once",
                color = PipePalette.TextPrimary,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )
            Spacer(modifier = Modifier.height(6.dp))
            Text(
                text = "The app now creates mobile users automatically. Use a clean username and save the server address once.",
                color = PipePalette.TextSecondary,
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(modifier = Modifier.height(16.dp))
            TipsRow()
            Spacer(modifier = Modifier.height(16.dp))
            OutlinedTextField(
                value = state.endpoint,
                onValueChange = onEndpointChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Server address") },
                placeholder = { Text("http://192.168.x.x:8080") },
                singleLine = true,
                colors = pipeTextFieldColors(),
            )
            Spacer(modifier = Modifier.height(12.dp))
            OutlinedTextField(
                value = state.usernameInput,
                onValueChange = onUsernameChange,
                modifier = Modifier.fillMaxWidth(),
                label = { Text("Username") },
                placeholder = { Text("inko_mobile") },
                singleLine = true,
                colors = pipeTextFieldColors(),
            )
            Spacer(modifier = Modifier.height(10.dp))
            Text(
                text = "Allowed: letters, numbers, `_` and `-`. Minimum 2 characters.",
                color = PipePalette.TextMuted,
                style = MaterialTheme.typography.bodySmall,
            )
            if (state.error != null) {
                Spacer(modifier = Modifier.height(14.dp))
                ErrorBanner(state.error)
            }
            Spacer(modifier = Modifier.height(18.dp))
            Button(
                onClick = onConnect,
                modifier = Modifier
                    .fillMaxWidth()
                    .defaultMinSize(minHeight = 54.dp),
                enabled = !state.loading,
                colors = ButtonDefaults.buttonColors(
                    containerColor = PipePalette.AccentBlue,
                    contentColor = PipePalette.TextPrimary,
                    disabledContainerColor = PipePalette.PanelSoft,
                    disabledContentColor = PipePalette.TextMuted,
                ),
                shape = RoundedCornerShape(20.dp),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = PipePalette.TextPrimary,
                    )
                } else {
                    Text("Open workspace")
                }
            }
        }
        Spacer(modifier = Modifier.height(14.dp))
        Text(
            text = "If the browser can open `/api/mobile/health` but the app still fails, install the latest APK from this branch.",
            color = PipePalette.TextMuted,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(modifier = Modifier.height(12.dp))
    }
}

@Composable
private fun ChatScreen(
    state: PipeUiState,
    onDraftChange: (String) -> Unit,
    onRefresh: () -> Unit,
    onShowMentions: () -> Unit,
    onOpenMembers: () -> Unit,
    onOpenModLog: () -> Unit,
    onDmInputChange: (String) -> Unit,
    onCreateDm: () -> Unit,
    onLookupUser: (String) -> Unit,
    onCloseSheet: () -> Unit,
    onChatClick: (String) -> Unit,
    onSelectMessage: (String) -> Unit,
    onStartReply: () -> Unit,
    onStartEdit: () -> Unit,
    onCancelEdit: () -> Unit,
    onDeleteSelected: () -> Unit,
    onSend: () -> Unit,
    onNewChannelNameChange: (String) -> Unit,
    onToggleNewChannelPrivacy: () -> Unit,
    onCreateChannel: () -> Unit,
    onMemberInputChange: (String) -> Unit,
    onInviteMember: () -> Unit,
    onRemoveMember: () -> Unit,
) {
    val selectedMessage = state.messages.firstOrNull { it.id == state.selectedMessageId }
    val replyTarget = state.messages.firstOrNull { it.id == state.replyingToMessageId }
    val canManageMembers = state.profile?.role == "owner" || state.profile?.role == "admin"

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(screenBackgroundBrush())
            .statusBarsPadding()
            .navigationBarsPadding(),
    ) {
        HeaderHero(
            state = state,
            onShowMentions = onShowMentions,
            onRefresh = onRefresh,
        )
        StatusStrip(state = state)
        if (state.profile?.role == "owner") {
            ChannelComposerCard(
                state = state,
                onNameChange = onNewChannelNameChange,
                onTogglePrivacy = onToggleNewChannelPrivacy,
                onCreate = onCreateChannel,
            )
        }
        QuickActions(
            state = state,
            onDmInputChange = onDmInputChange,
            onCreateDm = onCreateDm,
            onOpenMembers = onOpenMembers,
            onOpenModLog = onOpenModLog,
            onLookupUser = onLookupUser,
        )
        SectionHeader(
            title = if (state.showingMentions) "Inbox" else "Chats",
            subtitle = if (state.showingMentions) {
                "${state.messages.size} mentions loaded"
            } else {
                "${state.chats.size} rooms available"
            },
        )
        LazyRow(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 2.dp, bottom = 10.dp),
            contentPadding = PaddingValues(horizontal = 16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            items(state.chats, key = { it.id }) { chat ->
                ChatCard(
                    chat = chat,
                    selected = chat.id == state.selectedChatId && !state.showingMentions,
                    onClick = { onChatClick(chat.id) },
                )
            }
        }
        Surface(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .padding(horizontal = 16.dp),
            color = PipePalette.Panel,
            shape = RoundedCornerShape(30.dp),
            border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
        ) {
            MessageTimeline(
                messages = state.messages,
                selectedMessageId = state.selectedMessageId,
                onSelectMessage = onSelectMessage,
            )
        }
        if (state.activeSheet != SheetContent.None) {
            InfoSheet(
                state = state,
                canManageMembers = canManageMembers,
                onMemberInputChange = onMemberInputChange,
                onInviteMember = onInviteMember,
                onRemoveMember = onRemoveMember,
                onClose = onCloseSheet,
            )
        }
        ComposerDock(
            state = state,
            selectedMessage = selectedMessage,
            replyTarget = replyTarget,
            onDraftChange = onDraftChange,
            onStartReply = onStartReply,
            onStartEdit = onStartEdit,
            onDeleteSelected = onDeleteSelected,
            onCancelEdit = onCancelEdit,
            onSend = onSend,
        )
    }
}

@Composable
private fun HeaderHero(
    state: PipeUiState,
    onShowMentions: () -> Unit,
    onRefresh: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        color = Color.Transparent,
        shape = RoundedCornerShape(32.dp),
        border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.9f)),
    ) {
        Box(
            modifier = Modifier
                .background(
                    Brush.linearGradient(
                        listOf(
                            PipePalette.BackgroundElevated,
                            PipePalette.Panel,
                            PipePalette.PanelSoft,
                        ),
                    ),
                )
                .padding(18.dp),
        ) {
            Column {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.Top,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = state.profile?.username ?: "unknown",
                            color = PipePalette.TextPrimary,
                            style = MaterialTheme.typography.headlineSmall,
                            fontWeight = FontWeight.Black,
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = shortEndpoint(state.profile?.endpoint.orEmpty()),
                            color = PipePalette.TextSecondary,
                            style = MaterialTheme.typography.bodySmall,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    Spacer(modifier = Modifier.width(12.dp))
                    Column(horizontalAlignment = Alignment.End) {
                        SmallActionButton(
                            text = if (state.showingMentions) "Inbox live" else "Inbox",
                            onClick = onShowMentions,
                            accent = state.showingMentions,
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                        SmallActionButton(
                            text = if (state.loading) "Syncing" else "Sync",
                            onClick = onRefresh,
                            accent = false,
                        )
                    }
                }
                Spacer(modifier = Modifier.height(14.dp))
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    PipeChip(text = roleLabel(state.profile?.role.orEmpty()), background = PipePalette.SoftViolet, color = PipePalette.AccentLilac)
                    PipeChip(text = "${state.chats.size} chats", background = PipePalette.SoftCyan, color = PipePalette.AccentCyan)
                    PipeChip(text = "${state.messages.size} loaded", background = PipePalette.PanelRaised, color = PipePalette.TextSecondary)
                }
            }
        }
    }
}

@Composable
private fun StatusStrip(state: PipeUiState) {
    val profile = state.profile ?: return
    PipePanel(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(46.dp)
                    .clip(CircleShape)
                    .background(Brush.linearGradient(listOf(PipePalette.AccentCyan, PipePalette.AccentViolet))),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = profile.username.take(1).uppercase(),
                    color = PipePalette.TextPrimary,
                    fontWeight = FontWeight.Black,
                )
            }
            Spacer(modifier = Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = profile.bio.ifBlank { "Local chat without terminal friction." },
                    color = PipePalette.TextPrimary,
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = "Selected room: ${state.chats.firstOrNull { it.id == state.selectedChatId }?.title ?: "none"}",
                    color = PipePalette.TextMuted,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
private fun ChannelComposerCard(
    state: PipeUiState,
    onNameChange: (String) -> Unit,
    onTogglePrivacy: () -> Unit,
    onCreate: () -> Unit,
) {
    PipePanel(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 10.dp),
        accent = true,
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "Create channel",
                    color = PipePalette.TextPrimary,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = "Owner controls stay visible but no longer dominate the whole screen.",
                    color = PipePalette.TextSecondary,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            PipeChip(
                text = if (state.newChannelPrivate) "Private" else "Public",
                background = if (state.newChannelPrivate) PipePalette.SoftViolet else PipePalette.SoftCyan,
                color = if (state.newChannelPrivate) PipePalette.AccentLilac else PipePalette.AccentCyan,
            )
        }
        Spacer(modifier = Modifier.height(12.dp))
        OutlinedTextField(
            value = state.newChannelName,
            onValueChange = onNameChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Channel name") },
            placeholder = { Text("#launch-room") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            TextButton(
                onClick = onTogglePrivacy,
                colors = ButtonDefaults.textButtonColors(
                    contentColor = PipePalette.TextPrimary,
                    containerColor = PipePalette.PanelSoft,
                ),
                shape = RoundedCornerShape(18.dp),
            ) {
                Text(if (state.newChannelPrivate) "Make public" else "Make private")
            }
            Button(
                onClick = onCreate,
                enabled = state.newChannelName.isNotBlank() && !state.loading,
                colors = primaryButtonColors(),
                shape = RoundedCornerShape(18.dp),
            ) {
                Text("Create")
            }
        }
    }
}

@Composable
private fun QuickActions(
    state: PipeUiState,
    onDmInputChange: (String) -> Unit,
    onCreateDm: () -> Unit,
    onOpenMembers: () -> Unit,
    onOpenModLog: () -> Unit,
    onLookupUser: (String) -> Unit,
) {
    PipePanel(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
    ) {
        Text(
            text = "Quick actions",
            color = PipePalette.TextPrimary,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
        )
        Spacer(modifier = Modifier.height(6.dp))
        Text(
            text = "Search a user once, then jump straight into DM, profile lookup, people or moderation.",
            color = PipePalette.TextSecondary,
            style = MaterialTheme.typography.bodySmall,
        )
        Spacer(modifier = Modifier.height(14.dp))
        OutlinedTextField(
            value = state.quickDmInput,
            onValueChange = onDmInputChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("User focus") },
            placeholder = { Text("username") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            QuickActionButton(
                text = "DM",
                modifier = Modifier.weight(1f),
                enabled = state.quickDmInput.isNotBlank(),
                accent = true,
                onClick = onCreateDm,
            )
            QuickActionButton(
                text = "Profile",
                modifier = Modifier.weight(1f),
                enabled = state.quickDmInput.isNotBlank(),
                accent = false,
                onClick = { onLookupUser(state.quickDmInput) },
            )
        }
        Spacer(modifier = Modifier.height(10.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            QuickActionButton(
                text = "People",
                modifier = Modifier.weight(1f),
                enabled = true,
                accent = false,
                onClick = onOpenMembers,
            )
            QuickActionButton(
                text = "Log",
                modifier = Modifier.weight(1f),
                enabled = true,
                accent = false,
                onClick = onOpenModLog,
            )
        }
    }
}

@Composable
private fun SectionHeader(title: String, subtitle: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column {
            Text(
                text = title,
                color = PipePalette.TextPrimary,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )
            Text(
                text = subtitle,
                color = PipePalette.TextMuted,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

@Composable
private fun ChatCard(chat: ChatSummary, selected: Boolean, onClick: () -> Unit) {
    Surface(
        modifier = Modifier
            .width(196.dp)
            .clip(PipeCardShape)
            .clickable(onClick = onClick),
        color = if (selected) Color.Transparent else PipePalette.Panel,
        shape = PipeCardShape,
        border = BorderStroke(
            1.dp,
            if (selected) PipePalette.AccentCyan.copy(alpha = 0.9f) else PipePalette.Outline.copy(alpha = 0.8f),
        ),
    ) {
        Box(
            modifier = Modifier
                .background(
                    if (selected) {
                        Brush.linearGradient(
                            listOf(
                                PipePalette.PanelSoft,
                                PipePalette.BackgroundElevated,
                                PipePalette.Panel,
                            ),
                        )
                    } else {
                        Brush.linearGradient(listOf(PipePalette.Panel, PipePalette.PanelRaised))
                    },
                )
                .padding(16.dp),
        ) {
            Column {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = chat.title,
                        color = PipePalette.TextPrimary,
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    PipeChip(
                        text = if (chat.isPrivate) "Private" else "Open",
                        background = if (chat.isPrivate) PipePalette.SoftViolet else PipePalette.PanelSoft,
                        color = if (chat.isPrivate) PipePalette.AccentLilac else PipePalette.TextSecondary,
                    )
                }
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = chat.subtitle.ifBlank { "No topic yet." },
                    color = PipePalette.TextSecondary,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Spacer(modifier = Modifier.height(14.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (chat.unreadCount > 0) {
                        Badge(text = "${chat.unreadCount} new", background = PipePalette.SoftCyan, color = PipePalette.AccentCyan)
                    }
                    if (chat.mentionCount > 0) {
                        Badge(text = "@${chat.mentionCount}", background = PipePalette.SoftViolet, color = PipePalette.AccentLilac)
                    }
                    if (chat.unreadCount == 0 && chat.mentionCount == 0) {
                        Badge(text = "quiet", background = PipePalette.PanelSoft, color = PipePalette.TextMuted)
                    }
                }
            }
        }
    }
}

@Composable
private fun MessageTimeline(
    messages: List<MessageItem>,
    selectedMessageId: String?,
    onSelectMessage: (String) -> Unit,
) {
    if (messages.isEmpty()) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(20.dp),
            contentAlignment = Alignment.Center,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "No messages yet",
                    color = PipePalette.TextPrimary,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                )
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = "Pick a room or send the first message to kick the conversation off.",
                    color = PipePalette.TextSecondary,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 14.dp, vertical = 16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        items(messages, key = { it.id }) { message ->
            MessageBubble(
                message = message,
                selected = message.id == selectedMessageId,
                onClick = { onSelectMessage(message.id) },
            )
        }
    }
}

@Composable
private fun MessageBubble(
    message: MessageItem,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (message.isMine) Arrangement.End else Arrangement.Start,
    ) {
        Surface(
            modifier = Modifier
                .widthIn(max = 320.dp)
                .clickable(onClick = onClick),
            color = when {
                selected -> PipePalette.Selection
                message.isMine -> PipePalette.MineBubble
                else -> PipePalette.OtherBubble
            },
            shape = RoundedCornerShape(
                topStart = 24.dp,
                topEnd = 24.dp,
                bottomEnd = if (message.isMine) 10.dp else 24.dp,
                bottomStart = if (message.isMine) 24.dp else 10.dp,
            ),
            border = BorderStroke(
                1.dp,
                when {
                    selected -> PipePalette.AccentCyan
                    message.isMine -> PipePalette.AccentBlue.copy(alpha = 0.45f)
                    else -> PipePalette.Outline.copy(alpha = 0.7f)
                },
            ),
        ) {
            Column(modifier = Modifier.padding(14.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        text = if (message.isMine) "You" else message.author,
                        color = if (message.isMine) PipePalette.AccentCyan else PipePalette.TextPrimary,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.labelLarge,
                    )
                    Spacer(modifier = Modifier.width(12.dp))
                    Text(
                        text = buildString {
                            append(message.timeLabel)
                            if (message.isEdited) append(" · edited")
                        },
                        color = PipePalette.TextMuted,
                        style = MaterialTheme.typography.labelSmall,
                    )
                }
                if (message.replyPreview.isNotBlank()) {
                    Spacer(modifier = Modifier.height(10.dp))
                    Surface(
                        color = PipePalette.PanelRaised.copy(alpha = 0.95f),
                        shape = RoundedCornerShape(16.dp),
                        border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.7f)),
                    ) {
                        Text(
                            text = message.replyPreview,
                            modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
                            color = PipePalette.TextSecondary,
                            style = MaterialTheme.typography.bodySmall,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
                Spacer(modifier = Modifier.height(10.dp))
                Text(
                    text = message.body,
                    color = PipePalette.TextPrimary,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }
    }
}

@Composable
private fun ComposerDock(
    state: PipeUiState,
    selectedMessage: MessageItem?,
    replyTarget: MessageItem?,
    onDraftChange: (String) -> Unit,
    onStartReply: () -> Unit,
    onStartEdit: () -> Unit,
    onDeleteSelected: () -> Unit,
    onCancelEdit: () -> Unit,
    onSend: () -> Unit,
) {
    PipePanel(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        accent = true,
    ) {
        if (selectedMessage != null) {
            SelectedMessageBar(
                selectedMessage = selectedMessage,
                isEditing = state.editingMessageId != null,
                onReply = onStartReply,
                onEdit = onStartEdit,
                onDelete = onDeleteSelected,
                onCancel = onCancelEdit,
            )
            Spacer(modifier = Modifier.height(10.dp))
        }
        if (replyTarget != null || state.editingMessageId != null) {
            ReplyBanner(
                replyTarget = replyTarget,
                isEditing = state.editingMessageId != null,
                onCancel = onCancelEdit,
            )
            Spacer(modifier = Modifier.height(10.dp))
        }
        if (state.error != null) {
            ErrorBanner(state.error)
            Spacer(modifier = Modifier.height(10.dp))
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.Bottom,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            OutlinedTextField(
                value = state.composerText,
                onValueChange = onDraftChange,
                modifier = Modifier.weight(1f),
                label = { Text(if (state.editingMessageId != null) "Edit message" else "Message") },
                placeholder = { Text(if (state.editingMessageId != null) "Refine the selected message" else "Write something sharp") },
                shape = RoundedCornerShape(22.dp),
                colors = pipeTextFieldColors(),
            )
            Button(
                onClick = onSend,
                enabled = !state.loading && state.composerText.isNotBlank() && state.selectedChatId != null,
                modifier = Modifier
                    .defaultMinSize(minWidth = 96.dp, minHeight = 56.dp),
                colors = primaryButtonColors(),
                shape = RoundedCornerShape(20.dp),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                        color = PipePalette.TextPrimary,
                    )
                } else {
                    Text(if (state.editingMessageId != null) "Save" else "Send")
                }
            }
        }
    }
}

@Composable
private fun SelectedMessageBar(
    selectedMessage: MessageItem,
    isEditing: Boolean,
    onReply: () -> Unit,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
    onCancel: () -> Unit,
) {
    Surface(
        color = PipePalette.PanelRaised,
        shape = RoundedCornerShape(22.dp),
        border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(
                text = if (isEditing) "Editing message" else "Selected message",
                color = PipePalette.TextPrimary,
                fontWeight = FontWeight.Bold,
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = selectedMessage.body,
                color = PipePalette.TextSecondary,
                style = MaterialTheme.typography.bodySmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(modifier = Modifier.height(10.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                SmallTextPill("Reply", onReply)
                if (selectedMessage.isMine) {
                    SmallTextPill("Edit", onEdit)
                    SmallTextPill("Delete", onDelete, destructive = true)
                }
                SmallTextPill("Close", onCancel)
            }
        }
    }
}

@Composable
private fun ReplyBanner(
    replyTarget: MessageItem?,
    isEditing: Boolean,
    onCancel: () -> Unit,
) {
    Surface(
        color = PipePalette.PanelRaised,
        shape = RoundedCornerShape(20.dp),
        border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = if (isEditing) "Editing current draft" else "Replying to ${replyTarget?.author ?: "message"}",
                    color = PipePalette.AccentLilac,
                    fontWeight = FontWeight.Bold,
                )
                if (replyTarget != null && !isEditing) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = replyTarget.body,
                        color = PipePalette.TextSecondary,
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Spacer(modifier = Modifier.width(12.dp))
            SmallTextPill("Cancel", onCancel)
        }
    }
}

@Composable
private fun InfoSheet(
    state: PipeUiState,
    canManageMembers: Boolean,
    onMemberInputChange: (String) -> Unit,
    onInviteMember: () -> Unit,
    onRemoveMember: () -> Unit,
    onClose: () -> Unit,
) {
    PipePanel(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 10.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = when (state.activeSheet) {
                    SheetContent.Members -> "People"
                    SheetContent.UserLookup -> "Profile"
                    SheetContent.ModLog -> "Moderation log"
                    SheetContent.None -> ""
                },
                color = PipePalette.TextPrimary,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            SmallTextPill("Close", onClose)
        }
        Spacer(modifier = Modifier.height(10.dp))
        HorizontalDivider(color = PipePalette.Outline.copy(alpha = 0.7f))
        Spacer(modifier = Modifier.height(10.dp))
        when (state.activeSheet) {
            SheetContent.Members -> MemberSheet(
                members = state.members,
                canManageMembers = canManageMembers,
                memberActionInput = state.memberActionInput,
                onMemberInputChange = onMemberInputChange,
                onInviteMember = onInviteMember,
                onRemoveMember = onRemoveMember,
            )
            SheetContent.UserLookup -> UserCard(state.lookedUpUser)
            SheetContent.ModLog -> ModLogList(state.modLog)
            SheetContent.None -> Unit
        }
    }
}

@Composable
private fun MemberSheet(
    members: List<UserSummary>,
    canManageMembers: Boolean,
    memberActionInput: String,
    onMemberInputChange: (String) -> Unit,
    onInviteMember: () -> Unit,
    onRemoveMember: () -> Unit,
) {
    if (canManageMembers) {
        OutlinedTextField(
            value = memberActionInput,
            onValueChange = onMemberInputChange,
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Invite or remove user") },
            placeholder = { Text("username") },
            singleLine = true,
            colors = pipeTextFieldColors(),
        )
        Spacer(modifier = Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Button(
                onClick = onInviteMember,
                enabled = memberActionInput.isNotBlank(),
                colors = primaryButtonColors(),
                shape = RoundedCornerShape(18.dp),
            ) {
                Text("Invite")
            }
            TextButton(
                onClick = onRemoveMember,
                colors = ButtonDefaults.textButtonColors(
                    contentColor = PipePalette.Danger,
                    containerColor = PipePalette.SoftDanger,
                ),
                shape = RoundedCornerShape(18.dp),
            ) {
                Text("Remove")
            }
        }
        Spacer(modifier = Modifier.height(14.dp))
    }
    MemberList(members)
}

@Composable
private fun MemberList(members: List<UserSummary>) {
    if (members.isEmpty()) {
        Text("No members to show.", color = PipePalette.TextSecondary)
        return
    }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(max = 240.dp)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        members.forEach { member ->
            Surface(
                color = PipePalette.PanelRaised,
                shape = RoundedCornerShape(20.dp),
                border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(member.username, fontWeight = FontWeight.Bold, color = PipePalette.TextPrimary)
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(member.role, color = PipePalette.AccentLilac, style = MaterialTheme.typography.labelMedium)
                    if (member.bio.isNotBlank()) {
                        Spacer(modifier = Modifier.height(6.dp))
                        Text(member.bio, color = PipePalette.TextSecondary, style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
    }
}

@Composable
private fun UserCard(user: UserSummary?) {
    if (user == null) {
        Text("User not loaded.", color = PipePalette.TextSecondary)
        return
    }
    Surface(
        color = PipePalette.PanelRaised,
        shape = RoundedCornerShape(20.dp),
        border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            Text(user.username, fontWeight = FontWeight.Black, color = PipePalette.TextPrimary)
            Spacer(modifier = Modifier.height(4.dp))
            Text("Role: ${user.role}", color = PipePalette.AccentLilac)
            Spacer(modifier = Modifier.height(8.dp))
            Text(user.bio.ifBlank { "No bio set." }, color = PipePalette.TextSecondary)
        }
    }
}

@Composable
private fun ModLogList(items: List<ModLogItem>) {
    if (items.isEmpty()) {
        Text("Moderation log is empty.", color = PipePalette.TextSecondary)
        return
    }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(max = 240.dp)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        items.forEach { item ->
            Surface(
                color = PipePalette.PanelRaised,
                shape = RoundedCornerShape(20.dp),
                border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text("${item.actor} -> ${item.target}", fontWeight = FontWeight.Bold, color = PipePalette.TextPrimary)
                    Spacer(modifier = Modifier.height(4.dp))
                    Text("${item.action} on ${item.channel}", color = PipePalette.AccentLilac, style = MaterialTheme.typography.labelMedium)
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(item.details, color = PipePalette.TextSecondary, style = MaterialTheme.typography.bodySmall)
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(item.createdAt, color = PipePalette.TextMuted, style = MaterialTheme.typography.labelSmall)
                }
            }
        }
    }
}

@Composable
private fun PipeBrandMark(modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier.size(112.dp),
        color = Color.Transparent,
        shape = RoundedCornerShape(34.dp),
        border = BorderStroke(1.dp, PipePalette.OutlineBright.copy(alpha = 0.8f)),
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.radialGradient(
                        colors = listOf(
                            PipePalette.PanelSoft,
                            PipePalette.BackgroundElevated,
                            PipePalette.Panel,
                        ),
                    ),
                ),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = "PN",
                color = PipePalette.TextPrimary,
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Black,
            )
        }
    }
}

@Composable
private fun TipsRow() {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        TipChip(
            title = "Phone",
            text = "Use the PC LAN IP, for example `http://192.168.x.x:8080`.",
        )
        TipChip(
            title = "Emulator",
            text = "Use `http://10.0.2.2:8080` instead of localhost.",
        )
    }
}

@Composable
private fun TipChip(title: String, text: String) {
    Surface(
        color = PipePalette.PanelRaised,
        shape = RoundedCornerShape(20.dp),
        border = BorderStroke(1.dp, PipePalette.Outline.copy(alpha = 0.8f)),
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(title, color = PipePalette.TextPrimary, fontWeight = FontWeight.Bold)
            Spacer(modifier = Modifier.height(4.dp))
            Text(text, color = PipePalette.TextSecondary, style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
private fun PipePanel(
    modifier: Modifier = Modifier,
    accent: Boolean = false,
    content: @Composable ColumnScope.() -> Unit,
) {
    Surface(
        modifier = modifier,
        color = if (accent) PipePalette.PanelRaised else PipePalette.Panel,
        shape = PipePanelShape,
        border = BorderStroke(
            1.dp,
            if (accent) PipePalette.OutlineBright.copy(alpha = 0.85f) else PipePalette.Outline.copy(alpha = 0.8f),
        ),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            content = content,
        )
    }
}

@Composable
private fun SmallActionButton(
    text: String,
    onClick: () -> Unit,
    accent: Boolean,
) {
    TextButton(
        onClick = onClick,
        colors = ButtonDefaults.textButtonColors(
            contentColor = if (accent) PipePalette.TextPrimary else PipePalette.TextSecondary,
            containerColor = if (accent) PipePalette.AccentBlue.copy(alpha = 0.22f) else PipePalette.PanelRaised,
        ),
        shape = RoundedCornerShape(16.dp),
    ) {
        Text(text)
    }
}

@Composable
private fun QuickActionButton(
    text: String,
    modifier: Modifier = Modifier,
    enabled: Boolean,
    accent: Boolean,
    onClick: () -> Unit,
) {
    Button(
        onClick = onClick,
        modifier = modifier.defaultMinSize(minHeight = 48.dp),
        enabled = enabled,
        colors = if (accent) {
            primaryButtonColors()
        } else {
            ButtonDefaults.buttonColors(
                containerColor = PipePalette.PanelSoft,
                contentColor = PipePalette.TextPrimary,
                disabledContainerColor = PipePalette.PanelSoft.copy(alpha = 0.55f),
                disabledContentColor = PipePalette.TextMuted,
            )
        },
        shape = RoundedCornerShape(18.dp),
    ) {
        Text(text)
    }
}

@Composable
private fun SmallTextPill(
    text: String,
    onClick: () -> Unit,
    destructive: Boolean = false,
) {
    TextButton(
        onClick = onClick,
        colors = ButtonDefaults.textButtonColors(
            contentColor = if (destructive) PipePalette.Danger else PipePalette.TextPrimary,
            containerColor = if (destructive) PipePalette.SoftDanger else PipePalette.PanelSoft,
        ),
        shape = RoundedCornerShape(16.dp),
    ) {
        Text(text)
    }
}

@Composable
private fun PipeChip(
    text: String,
    background: Color,
    color: Color,
) {
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(background)
            .padding(horizontal = 10.dp, vertical = 6.dp),
    ) {
        Text(
            text = text,
            color = color,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

@Composable
private fun Badge(
    text: String,
    background: Color,
    color: Color,
) {
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(background)
            .padding(horizontal = 9.dp, vertical = 5.dp),
    ) {
        Text(
            text = text,
            color = color,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

@Composable
private fun ErrorBanner(message: String) {
    Surface(
        color = PipePalette.SoftDanger,
        shape = RoundedCornerShape(18.dp),
        border = BorderStroke(1.dp, PipePalette.Danger.copy(alpha = 0.5f)),
    ) {
        Text(
            text = message,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            color = PipePalette.Danger,
            style = MaterialTheme.typography.bodySmall,
        )
    }
}

@Composable
private fun primaryButtonColors() = ButtonDefaults.buttonColors(
    containerColor = PipePalette.AccentBlue,
    contentColor = PipePalette.TextPrimary,
    disabledContainerColor = PipePalette.PanelSoft.copy(alpha = 0.55f),
    disabledContentColor = PipePalette.TextMuted,
)

@Composable
private fun pipeTextFieldColors() = OutlinedTextFieldDefaults.colors(
    focusedTextColor = PipePalette.TextPrimary,
    unfocusedTextColor = PipePalette.TextPrimary,
    disabledTextColor = PipePalette.TextMuted,
    focusedLabelColor = PipePalette.AccentCyan,
    unfocusedLabelColor = PipePalette.TextSecondary,
    focusedPlaceholderColor = PipePalette.TextMuted,
    unfocusedPlaceholderColor = PipePalette.TextMuted,
    focusedBorderColor = PipePalette.AccentBlue,
    unfocusedBorderColor = PipePalette.OutlineBright,
    cursorColor = PipePalette.AccentCyan,
    focusedContainerColor = PipePalette.PanelSoft.copy(alpha = 0.55f),
    unfocusedContainerColor = PipePalette.PanelSoft.copy(alpha = 0.4f),
)

private fun screenBackgroundBrush(): Brush = Brush.verticalGradient(
    listOf(
        PipePalette.Background,
        PipePalette.BackgroundElevated,
        PipePalette.Panel,
    ),
)

private fun roleLabel(role: String): String = when (role.lowercase()) {
    "owner" -> "Owner"
    "admin" -> "Admin"
    "user" -> "Member"
    else -> role.ifBlank { "Member" }
}

private fun shortEndpoint(endpoint: String): String {
    if (endpoint.length <= 34) return endpoint
    return endpoint.take(34) + "..."
}
