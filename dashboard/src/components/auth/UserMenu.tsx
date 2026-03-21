import { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '../../stores/authStore';

export default function UserMenu() {
  const { t } = useTranslation();
  const { user, logout } = useAuthStore();
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  if (!user) return null;

  return (
    <div className="user-menu" ref={menuRef}>
      <button className="user-menu-trigger" onClick={() => setIsOpen(!isOpen)}>
        <div className="user-avatar">
          {user.username.charAt(0).toUpperCase()}
        </div>
        <div className="user-info">
          <span className="user-name">{user.username}</span>
          <span className="user-tenant">{user.tenantName}</span>
        </div>
      </button>
      {isOpen && (
        <div className="user-menu-dropdown">
          <div className="user-menu-header">
            <span className="user-menu-role">{user.role}</span>
            <span className="user-menu-email">{user.email}</span>
          </div>
          <div className="user-menu-divider" />
          <button className="user-menu-item" onClick={logout}>
            {t('auth.signOut')}
          </button>
        </div>
      )}
    </div>
  );
}
