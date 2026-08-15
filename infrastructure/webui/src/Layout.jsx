import { useState, useEffect, useCallback } from "react";
import PropTypes from "prop-types";
import { Link, Outlet, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { clearAuth, getAuth, isAuthenticated } from "./utils/auth";
import {
  LayoutDashboard,
  FolderOpen,
  Database,
  Settings,
  LogOut,
  Lock,
  Unlock,
  Bell,
  CloudLightning,
  Menu,
  X,
  Activity,
  User,
  KeyRound,
  Shield,
  FolderSync,
  Globe,
  HardDrive,
  FileText,
  Palette,
  Users,
  Info,
  Home,
  ChevronLeft
} from "lucide-react";
import { ThemeToggle } from "./components/ThemeToggle";
import { PowerStatusBanner } from "./components/PowerStatusBanner";
import { useSwipe } from "./hooks/useSwipe";
import { useVault } from "./context/VaultContext";
import { VaultModal } from "./components/Vault/VaultModal";

// SidebarItem Component for reusable navigation items
const SidebarItem = ({ icon: Icon, label, active, onClick }) => (
  <button
    onClick={onClick}
    className={`flex items-center gap-3 w-full p-3 rounded-xl transition-all duration-300 group
      ${active
        ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30 shadow-[0_0_15px_rgba(59,130,246,0.3)]'
        : 'text-slate-400 hover:bg-white/5 hover:text-white hover:pl-4'
      }`}
  >
    <Icon size={20} strokeWidth={1.5} />
    <span className="font-medium text-sm tracking-wide">{label}</span>
    {active && <div className="ml-auto w-1 h-1 bg-blue-400 rounded-full shadow-[0_0_8px_rgba(59,130,246,0.8)]" />}
  </button>
);

SidebarItem.propTypes = {
  icon: PropTypes.elementType.isRequired,
  label: PropTypes.string.isRequired,
  active: PropTypes.bool.isRequired,
  onClick: PropTypes.func.isRequired,
};

export default function Layout({ title = "NAS AI v1.0.0" }) {
  const location = useLocation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { vaultConfig } = useVault(); // Read vault config to check if setup is needed
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  // Vault Global State
  const { isUnlocked, unlock, setup } = useVault();
  const [showVaultModal, setShowVaultModal] = useState(false);

  // Vault Handlers
  const handleVaultUnlock = async (password) => {
    await unlock(password);
    setShowVaultModal(false);
  };

  const handleVaultSetup = async (data) => {
    await setup(data);
    setShowVaultModal(false);
  };

  // Check if we're on settings page
  const isSettingsPage = location.pathname === '/settings';
  const activeSettingsTab = searchParams.get('tab') || 'profile';

  // Scroll effect and Event Listeners
  useEffect(() => {
    // Scroll listener
    const handleScroll = () => setScrolled(window.scrollY > 20);
    window.addEventListener('scroll', handleScroll);

    // Vault Locked Listener (Global Modal Trigger)
    const handleLockedEvent = () => setShowVaultModal(true);
    window.addEventListener('vault-locked', handleLockedEvent);

    return () => {
      window.removeEventListener('scroll', handleScroll);
      window.removeEventListener('vault-locked', handleLockedEvent);
    };
  }, []);

  // Swipe handlers for sidebar
  const closeSidebar = useCallback(() => setMobileMenuOpen(false), []);
  const openSidebar = useCallback(() => setMobileMenuOpen(true), []);

  const sidebarSwipeProps = useSwipe(closeSidebar, null, { threshold: 50 });
  const edgeSwipeProps = useSwipe(null, openSidebar, { threshold: 30 });

  const handleLogout = () => {
    clearAuth();
    navigate("/login", { replace: true });
  };

  // Auth guard: redirect unauthenticated users to login
  useEffect(() => {
    if (!isAuthenticated() && location.pathname !== "/login") {
      navigate("/login", { replace: true, state: { from: location.pathname } });
    }
  }, [location.pathname, navigate]);

  // Keyboard shortcuts (Cmd/Ctrl + B = Backups, Cmd/Ctrl + D = Dashboard)
  useEffect(() => {
    const handleKeyDown = (e) => {
      // Ignore if user is typing in input
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

      const isMod = e.metaKey || e.ctrlKey;
      if (!isMod) return;

      switch (e.key.toLowerCase()) {
        case 'b':
          e.preventDefault();
          navigate('/backups');
          break;
        case 'd':
          e.preventDefault();
          navigate('/dashboard');
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [navigate]);


  const navLinks = [
    { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
    { to: "/metrics", label: "Metrics", icon: Activity },
    { to: "/files", label: "Files & Storage", icon: FolderOpen },
    { to: "/files/vault", label: "Secure Vault", icon: Lock },
    { to: "/backups", label: "Backups", icon: Database },
  ];

  // Settings tabs - shown in sidebar when on /settings page
  const settingsTabs = [
    { id: "profile", label: "Profil", icon: User },
    { id: "crypto", label: "Crypto / Files", icon: KeyRound },
    { id: "security", label: "Security", icon: Shield },
    { id: "backup", label: "Backup", icon: FolderSync },
    { id: "network", label: "Netzwerk", icon: Globe },
    { id: "storage", label: "Speicher", icon: HardDrive },
    { id: "logs", label: "Logs", icon: FileText },
    { id: "appearance", label: "Erscheinung", icon: Palette },
    { id: "admin", label: "Admin", icon: Users },
    { id: "about", label: "Über", icon: Info }
  ];

  const handleSettingsTabClick = (tabId) => {
    setSearchParams({ tab: tabId });
    setMobileMenuOpen(false);
  };

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans selection:bg-blue-500/30 overflow-x-hidden relative">

      {/* Background Effects (Liquid/Glow) */}
      <div className="fixed inset-0 z-0 pointer-events-none overflow-hidden">
        {/* Blue Blob */}
        <div className="absolute top-[-10%] left-[-10%] w-[500px] h-[500px] bg-blue-600/20 rounded-full blur-[120px] animate-pulse-glow"></div>
        {/* Violet Blob */}
        <div className="absolute bottom-[-10%] right-[-5%] w-[600px] h-[600px] bg-violet-600/10 rounded-full blur-[130px]"></div>
        {/* Cyan Accent middle */}
        <div className="absolute top-[40%] left-[30%] w-[300px] h-[300px] bg-cyan-500/10 rounded-full blur-[100px] opacity-60"></div>
      </div>

      <div className="relative z-10 flex h-screen overflow-hidden">

        {/* Mobile Overlay/Backdrop */}
        {mobileMenuOpen && (
          <div
            className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm lg:hidden"
            onClick={closeSidebar}
            {...edgeSwipeProps}
          />
        )}

        {/* Sidebar (Desktop & Mobile) */}
        <aside
          className={`
            fixed inset-y-0 left-0 z-50 w-72 transform transition-transform duration-300 ease-in-out
            lg:relative lg:translate-x-0
            ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}
            bg-[#0a0a0c]/95 backdrop-blur-2xl border-r border-white/5 flex flex-col
          `}
          {...sidebarSwipeProps}
        >
          {/* Logo Area */}
          <div className="p-8 pb-4 flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-violet-600 flex items-center justify-center shadow-lg shadow-blue-500/20">
              <CloudLightning size={22} className="text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-white tracking-wide">{title}</h1>
              <p className="text-[10px] text-blue-400 font-medium tracking-widest uppercase">System Online</p>
            </div>
          </div>

          {/* Navigation - Changes based on current page */}
          <nav className="flex-1 px-4 py-6 space-y-2 overflow-y-auto">
            {isSettingsPage ? (
              <>
                {/* Settings Navigation */}
                <p className="px-4 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Einstellungen</p>
                {settingsTabs.map((tab) => {
                  const active = activeSettingsTab === tab.id;
                  return (
                    <SidebarItem
                      key={tab.id}
                      icon={tab.icon}
                      label={tab.label}
                      active={active}
                      onClick={() => handleSettingsTabClick(tab.id)}
                    />
                  );
                })}
              </>
            ) : (
              <>
                {/* Main Navigation */}
                <p className="px-4 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Main Menu</p>
                {navLinks.map((link) => {
                  const active = location.pathname.startsWith(link.to);
                  return (
                    <Link key={link.to} to={link.to} className="block">
                      <SidebarItem
                        icon={link.icon}
                        label={link.label}
                        active={active}
                        onClick={() => setMobileMenuOpen(false)}
                      />
                    </Link>
                  );
                })}

                <p className="px-4 text-xs font-semibold text-slate-500 uppercase tracking-wider mt-8 mb-2">Preferences</p>
                <SidebarItem
                  icon={Settings}
                  label="Settings"
                  active={location.pathname === '/settings'}
                  onClick={() => {
                    navigate('/settings');
                    setMobileMenuOpen(false);
                  }}
                />
              </>
            )}
          </nav>

          {/* Bottom Actions - Home Button when in settings, otherwise User & Logout */}
          <div className="p-4 border-t border-white/5 bg-gradient-to-t from-black/40 to-transparent">
            {isSettingsPage ? (
              <button
                onClick={() => navigate('/dashboard')}
                className="flex items-center gap-3 w-full p-3 rounded-xl text-blue-400 hover:bg-blue-500/10 transition-colors group"
              >
                <ChevronLeft size={20} className="group-hover:-translate-x-1 transition-transform" />
                <Home size={20} />
                <span className="font-medium text-sm">Zurück zum Dashboard</span>
              </button>
            ) : (
              <>
                {isAuthenticated() ? (
                  <>
                    <div className="flex items-center gap-3 p-3 rounded-xl bg-white/5 border border-white/10 mb-3">
                      <div className="w-8 h-8 rounded-full bg-slate-700 overflow-hidden border border-white/10">
                        <div className="w-full h-full bg-gradient-to-tr from-slate-600 to-slate-500 flex items-center justify-center text-xs text-white font-bold">
                          AU
                        </div>
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-white truncate">User</p>
                        <p className="text-xs text-slate-400 truncate">Admin Access</p>
                      </div>
                    </div>
                    <button
                      onClick={handleLogout}
                      className="flex items-center gap-3 w-full p-3 rounded-xl text-rose-400 hover:bg-rose-500/10 transition-colors group"
                    >
                      <LogOut size={20} className="group-hover:translate-x-1 transition-transform" />
                      <span className="font-medium text-sm">Logout</span>
                    </button>
                  </>
                ) : (
                  <Link to="/login" className="flex items-center gap-3 w-full p-3 rounded-xl text-blue-400 hover:bg-blue-500/10 transition-colors">
                    <span className="font-medium text-sm">Login</span>
                  </Link>
                )}
              </>
            )}
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 h-full overflow-y-auto relative scroll-smooth">
          {/* Header */}
          <header className={`sticky top-0 z-40 px-6 py-4 flex items-center justify-between transition-all duration-300 ${scrolled ? 'bg-[#0a0a0c]/80 backdrop-blur-md border-b border-white/5' : ''}`}>
            <div className="flex items-center gap-4 lg:hidden">
              <button onClick={() => setMobileMenuOpen(!mobileMenuOpen)} className="p-2 text-slate-400 hover:text-white">
                {mobileMenuOpen ? <X size={24} /> : <Menu size={24} />}
              </button>
              <span className="text-lg font-bold text-white">{title}</span>
            </div>

            <div className="hidden lg:block">
              <h2 className="text-2xl font-semibold text-white">
                {location.pathname === '/dashboard' && 'Dashboard Overview'}
                {location.pathname === '/metrics' && 'System Metrics'}
                {location.pathname === '/files' && 'Files & Storage'}
                {location.pathname === '/backups' && 'Backups'}
                {location.pathname === '/settings' && 'Settings'}
              </h2>
              <p className="text-slate-400 text-sm">Welcome back to your neural hub.</p>
            </div>

            <div className="flex items-center gap-4">
              {/* Vault Indicator */}
              <button
                onClick={() => setShowVaultModal(true)}
                className={`relative p-2.5 rounded-full border transition-all ${isUnlocked
                  ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400 hover:bg-emerald-500/20'
                  : 'bg-rose-500/10 border-rose-500/20 text-rose-400 hover:bg-rose-500/20'
                  }`}
                title={`Vault Status: ${isUnlocked ? 'Unlocked' : 'Locked'}`}
              >
                {isUnlocked ? <Unlock size={20} /> : <Lock size={20} />}
              </button>

              <button className="relative p-2.5 rounded-full bg-white/5 border border-white/10 text-slate-300 hover:bg-white/10 transition-colors">
                <Bell size={20} />
                <span className="absolute top-2 right-2 w-2 h-2 bg-rose-500 rounded-full shadow-[0_0_8px_rgba(244,63,94,0.6)]"></span>
              </button>
              <ThemeToggle />
            </div>
          </header>

          {/* Page Content */}
          <div className="p-6 lg:p-10 max-w-[1600px] mx-auto">
            <PowerStatusBanner />
            <Outlet />
          </div>
        </main>
      </div>

      {/* Global Vault Modal */}
      <VaultModal
        isOpen={showVaultModal}
        onClose={() => setShowVaultModal(false)}
        onUnlock={handleVaultUnlock}
        onSetup={handleVaultSetup}
        hasVault={!!vaultConfig}
      />
    </div>
  );
}

Layout.propTypes = {
  title: PropTypes.string,
};
